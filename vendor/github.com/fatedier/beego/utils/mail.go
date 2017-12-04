// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	maxLineLength = 76

	upperhex = "0123456789ABCDEF"
)

// Email is the type used for email messages
type Email struct {
	Auth        smtp.Auth
	Identity    string `json:"identity"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	From        string `json:"from"`
	To          []string
	Bcc         []string
	Cc          []string
	Subject     string
	Text        string // Plaintext message (optional)
	HTML        string // Html message (optional)
	Headers     textproto.MIMEHeader
	Attachments []*Attachment
	ReadReceipt []string
}

// Attachment is a struct representing an email attachment.
// Based on the mime/multipart.FileHeader struct, Attachment contains the name, MIMEHeader, and content of the attachment in question
type Attachment struct {
	Filename string
	Header   textproto.MIMEHeader
	Content  []byte
}

// NewEMail create new Email struct with config json.
// config json is followed from Email struct fields.
func NewEMail(config string) *Email {
	e := new(Email)
	e.Headers = textproto.MIMEHeader{}
	err := json.Unmarshal([]byte(config), e)
	if err != nil {
		return nil
	}
	return e
}

// Bytes Make all send information to byte
func (e *Email) Bytes() ([]byte, error) {
	buff := &bytes.Buffer{}
	w := multipart.NewWriter(buff)
	// Set the appropriate headers (overwriting any conflicts)
	// Leave out Bcc (only included in envelope headers)
	e.Headers.Set("To", strings.Join(e.To, ","))
	if e.Cc != nil {
		e.Headers.Set("Cc", strings.Join(e.Cc, ","))
	}
	e.Headers.Set("From", e.From)
	e.Headers.Set("Subject", e.Subject)
	if len(e.ReadReceipt) != 0 {
		e.Headers.Set("Disposition-Notification-To", strings.Join(e.ReadReceipt, ","))
	}
	e.Headers.Set("MIME-Version", "1.0")

	// Write the envelope headers (including any custom headers)
	if err := headerToBytes(buff, e.Headers); err != nil {
		return nil, fmt.Errorf("Failed to render message headers: %s", err)
	}

	e.Headers.Set("Content-Type", fmt.Sprintf("multipart/mixed;\r\n boundary=%s\r\n", w.Boundary()))
	fmt.Fprintf(buff, "%s:", "Content-Type")
	fmt.Fprintf(buff, " %s\r\n", fmt.Sprintf("multipart/mixed;\r\n boundary=%s\r\n", w.Boundary()))

	// Start the multipart/mixed part
	fmt.Fprintf(buff, "--%s\r\n", w.Boundary())
	header := textproto.MIMEHeader{}
	// Check to see if there is a Text or HTML field
	if e.Text != "" || e.HTML != "" {
		subWriter := multipart.NewWriter(buff)
		// Create the multipart alternative part
		header.Set("Content-Type", fmt.Sprintf("multipart/alternative;\r\n boundary=%s\r\n", subWriter.Boundary()))
		// Write the header
		if err := headerToBytes(buff, header); err != nil {
			return nil, fmt.Errorf("Failed to render multipart message headers: %s", err)
		}
		// Create the body sections
		if e.Text != "" {
			header.Set("Content-Type", fmt.Sprintf("text/plain; charset=UTF-8"))
			header.Set("Content-Transfer-Encoding", "quoted-printable")
			if _, err := subWriter.CreatePart(header); err != nil {
				return nil, err
			}
			// Write the text
			if err := quotePrintEncode(buff, e.Text); err != nil {
				return nil, err
			}
		}
		if e.HTML != "" {
			header.Set("Content-Type", fmt.Sprintf("text/html; charset=UTF-8"))
			header.Set("Content-Transfer-Encoding", "quoted-printable")
			if _, err := subWriter.CreatePart(header); err != nil {
				return nil, err
			}
			// Write the text
			if err := quotePrintEncode(buff, e.HTML); err != nil {
				return nil, err
			}
		}
		if err := subWriter.Close(); err != nil {
			return nil, err
		}
	}
	// Create attachment part, if necessary
	for _, a := range e.Attachments {
		ap, err := w.CreatePart(a.Header)
		if err != nil {
			return nil, err
		}
		// Write the base64Wrapped content to the part
		base64Wrap(ap, a.Content)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// AttachFile Add attach file to the send mail
func (e *Email) AttachFile(args ...string) (a *Attachment, err error) {
	if len(args) < 1 && len(args) > 2 {
		err = errors.New("Must specify a file name and number of parameters can not exceed at least two")
		return
	}
	filename := args[0]
	id := ""
	if len(args) > 1 {
		id = args[1]
	}
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	ct := mime.TypeByExtension(filepath.Ext(filename))
	basename := path.Base(filename)
	return e.Attach(f, basename, ct, id)
}

// Attach is used to attach content from an io.Reader to the email.
// Parameters include an io.Reader, the desired filename for the attachment, and the Content-Type.
func (e *Email) Attach(r io.Reader, filename string, args ...string) (a *Attachment, err error) {
	if len(args) < 1 && len(args) > 2 {
		err = errors.New("Must specify the file type and number of parameters can not exceed at least two")
		return
	}
	c := args[0] //Content-Type
	id := ""
	if len(args) > 1 {
		id = args[1] //Content-ID
	}
	var buffer bytes.Buffer
	if _, err = io.Copy(&buffer, r); err != nil {
		return
	}
	at := &Attachment{
		Filename: filename,
		Header:   textproto.MIMEHeader{},
		Content:  buffer.Bytes(),
	}
	// Get the Content-Type to be used in the MIMEHeader
	if c != "" {
		at.Header.Set("Content-Type", c)
	} else {
		// If the Content-Type is blank, set the Content-Type to "application/octet-stream"
		at.Header.Set("Content-Type", "application/octet-stream")
	}
	if id != "" {
		at.Header.Set("Content-Disposition", fmt.Sprintf("inline;\r\n filename=\"%s\"", filename))
		at.Header.Set("Content-ID", fmt.Sprintf("<%s>", id))
	} else {
		at.Header.Set("Content-Disposition", fmt.Sprintf("attachment;\r\n filename=\"%s\"", filename))
	}
	at.Header.Set("Content-Transfer-Encoding", "base64")
	e.Attachments = append(e.Attachments, at)
	return at, nil
}

// Send will send out the mail
func (e *Email) Send() error {
	if e.Auth == nil {
		e.Auth = smtp.PlainAuth(e.Identity, e.Username, e.Password, e.Host)
	}
	// Merge the To, Cc, and Bcc fields
	to := make([]string, 0, len(e.To)+len(e.Cc)+len(e.Bcc))
	to = append(append(append(to, e.To...), e.Cc...), e.Bcc...)
	// Check to make sure there is at least one recipient and one "From" address
	if len(to) == 0 {
		return errors.New("Must specify at least one To address")
	}

	// Use the username if no From is provided
	if len(e.From) == 0 {
		e.From = e.Username
	}

	from, err := mail.ParseAddress(e.From)
	if err != nil {
		return err
	}

	// use mail's RFC 2047 to encode any string
	e.Subject = qEncode("utf-8", e.Subject)

	raw, err := e.Bytes()
	if err != nil {
		return err
	}
	return smtp.SendMail(e.Host+":"+strconv.Itoa(e.Port), e.Auth, from.Address, to, raw)
}

// quotePrintEncode writes the quoted-printable text to the IO Writer (according to RFC 2045)
func quotePrintEncode(w io.Writer, s string) error {
	var buf [3]byte
	mc := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		// We're assuming Unix style text formats as input (LF line break), and
		// quoted-printble uses CRLF line breaks. (Literal CRs will become
		// "=0D", but probably shouldn't be there to begin with!)
		if c == '\n' {
			io.WriteString(w, "\r\n")
			mc = 0
			continue
		}

		var nextOut []byte
		if isPrintable(c) {
			nextOut = append(buf[:0], c)
		} else {
			nextOut = buf[:]
			qpEscape(nextOut, c)
		}

		// Add a soft line break if the next (encoded) byte would push this line
		// to or past the limit.
		if mc+len(nextOut) >= maxLineLength {
			if _, err := io.WriteString(w, "=\r\n"); err != nil {
				return err
			}
			mc = 0
		}

		if _, err := w.Write(nextOut); err != nil {
			return err
		}
		mc += len(nextOut)
	}
	// No trailing end-of-line?? Soft line break, then. TODO: is this sane?
	if mc > 0 {
		io.WriteString(w, "=\r\n")
	}
	return nil
}

// isPrintable returns true if the rune given is "printable" according to RFC 2045, false otherwise
func isPrintable(c byte) bool {
	return (c >= '!' && c <= '<') || (c >= '>' && c <= '~') || (c == ' ' || c == '\n' || c == '\t')
}

// qpEscape is a helper function for quotePrintEncode which escapes a
// non-printable byte. Expects len(dest) == 3.
func qpEscape(dest []byte, c byte) {
	const nums = "0123456789ABCDEF"
	dest[0] = '='
	dest[1] = nums[(c&0xf0)>>4]
	dest[2] = nums[(c & 0xf)]
}

// headerToBytes enumerates the key and values in the header, and writes the results to the IO Writer
func headerToBytes(w io.Writer, t textproto.MIMEHeader) error {
	for k, v := range t {
		// Write the header key
		_, err := fmt.Fprintf(w, "%s:", k)
		if err != nil {
			return err
		}
		// Write each value in the header
		for _, c := range v {
			_, err := fmt.Fprintf(w, " %s\r\n", c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// base64Wrap encodes the attachment content, and wraps it according to RFC 2045 standards (every 76 chars)
// The output is then written to the specified io.Writer
func base64Wrap(w io.Writer, b []byte) {
	// 57 raw bytes per 76-byte base64 line.
	const maxRaw = 57
	// Buffer for each line, including trailing CRLF.
	var buffer [maxLineLength + len("\r\n")]byte
	copy(buffer[maxLineLength:], "\r\n")
	// Process raw chunks until there's no longer enough to fill a line.
	for len(b) >= maxRaw {
		base64.StdEncoding.Encode(buffer[:], b[:maxRaw])
		w.Write(buffer[:])
		b = b[maxRaw:]
	}
	// Handle the last chunk of bytes.
	if len(b) > 0 {
		out := buffer[:base64.StdEncoding.EncodedLen(len(b))]
		base64.StdEncoding.Encode(out, b)
		out = append(out, "\r\n"...)
		w.Write(out)
	}
}

// Encode returns the encoded-word form of s. If s is ASCII without special
// characters, it is returned unchanged. The provided charset is the IANA
// charset name of s. It is case insensitive.
// RFC 2047 encoded-word
func qEncode(charset, s string) string {
	if !needsEncoding(s) {
		return s
	}
	return encodeWord(charset, s)
}

func needsEncoding(s string) bool {
	for _, b := range s {
		if (b < ' ' || b > '~') && b != '\t' {
			return true
		}
	}
	return false
}

// encodeWord encodes a string into an encoded-word.
func encodeWord(charset, s string) string {
	buf := getBuffer()

	buf.WriteString("=?")
	buf.WriteString(charset)
	buf.WriteByte('?')
	buf.WriteByte('q')
	buf.WriteByte('?')

	enc := make([]byte, 3)
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b == ' ':
			buf.WriteByte('_')
		case b <= '~' && b >= '!' && b != '=' && b != '?' && b != '_':
			buf.WriteByte(b)
		default:
			enc[0] = '='
			enc[1] = upperhex[b>>4]
			enc[2] = upperhex[b&0x0f]
			buf.Write(enc)
		}
	}
	buf.WriteString("?=")

	es := buf.String()
	putBuffer(buf)
	return es
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Len() > 1024 {
		return
	}
	buf.Reset()
	bufPool.Put(buf)
}
