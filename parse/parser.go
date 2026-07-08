package parse

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/tool"
)

type Result struct {
	URL  *url.URL
	M3u8 *M3u8
	Keys map[int]crypt.KeyMaterial
}

func FromURL(link string, httpCfg *tool.HTTPConfig, cryptSvc *crypt.Service) (*Result, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	link = u.String()
	body, err := tool.Get(link, httpCfg)
	if err != nil {
		return nil, fmt.Errorf("request m3u8 URL failed: %s", err.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer body.Close()
	m3u8, err := parse(body)
	if err != nil {
		return nil, err
	}
	if len(m3u8.MasterPlaylist) != 0 {
		sf := m3u8.MasterPlaylist[0]
		return FromURL(tool.ResolveURL(u, sf.URI), httpCfg, cryptSvc)
	}
	if len(m3u8.Segments) == 0 {
		return nil, errors.New("can not found any TS file description")
	}
	result := &Result{
		URL:  u,
		M3u8: m3u8,
		Keys: make(map[int]crypt.KeyMaterial),
	}

	if cryptSvc != nil {
		var keyURIs []string
		for _, key := range m3u8.Keys {
			if key == nil || key.Method == "" || key.Method == CryptMethodNONE || key.URI == "" {
				continue
			}
			keyURIs = append(keyURIs, key.URI)
		}
		if id := cryptSvc.DetectProviderFromKeyURIs(keyURIs); id != "" {
			if err := cryptSvc.SetActiveProvider(id); err != nil {
				return nil, err
			}
		}
	}

	for idx, key := range m3u8.Keys {
		if key.Method == "" || key.Method == CryptMethodNONE {
			continue
		}
		if cryptSvc == nil && key.Method != CryptMethodAES {
			return nil, fmt.Errorf("unknown or unsupported cryption method: %s", key.Method)
		}
		keyURL := tool.ResolveURL(u, key.URI)
		resp, err := tool.Get(keyURL, httpCfg)
		if err != nil {
			return nil, fmt.Errorf("extract key failed: %s", err.Error())
		}
		keyByte, err := io.ReadAll(resp)
		_ = resp.Close()
		if err != nil {
			return nil, err
		}
		if cryptSvc != nil {
			meta := &crypt.KeyMeta{
				Method: string(key.Method),
				URI:    key.URI,
				IV:     key.IV,
			}
			ctx := &crypt.Context{
				M3U8URL: link,
				Method:  string(key.Method),
				KeyMeta: *meta,
			}
			material, err := cryptSvc.ProcessKey(ctx, keyByte, meta)
			if err != nil {
				return nil, err
			}
			result.Keys[idx] = material
		} else {
			iv := []byte(nil)
			if key.IV != "" {
				var ivErr error
				iv, ivErr = crypt.IVFromMeta(&crypt.KeyMeta{IV: key.IV})
				if ivErr != nil {
					return nil, ivErr
				}
			}
			result.Keys[idx] = crypt.KeyMaterial{Key: keyByte, IV: iv}
		}
	}
	return result, nil
}
