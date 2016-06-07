// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import "net/http"

type Data struct {
	ContentType string
	Data        []byte
}

func (r Data) Render(w http.ResponseWriter) error {
	if len(r.ContentType) > 0 {
		w.Header()["Content-Type"] = []string{r.ContentType}
	}
	w.Write(r.Data)
	return nil
}
