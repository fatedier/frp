// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import "log"

func (c *Context) GetCookie(name string) (string, error) {
	log.Println("GetCookie() method is deprecated. Use Cookie() instead.")
	return c.Cookie(name)
}
