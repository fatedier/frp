// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dnsmessage provides a mostly RFC 1035 compliant implementation of
// DNS message packing and unpacking.
//
// This implementation is designed to minimize heap allocations and avoid
// unnecessary packing and unpacking as much as possible.
package dnsmessage

import (
	"errors"
)

// Packet formats

// A Type is a type of DNS request and response.
type Type uint16

// A Class is a type of network.
type Class uint16

// An OpCode is a DNS operation code.
type OpCode uint16

// An RCode is a DNS response status code.
type RCode uint16

// Wire constants.
const (
	// ResourceHeader.Type and Question.Type
	TypeA     Type = 1
	TypeNS    Type = 2
	TypeCNAME Type = 5
	TypeSOA   Type = 6
	TypePTR   Type = 12
	TypeMX    Type = 15
	TypeTXT   Type = 16
	TypeAAAA  Type = 28
	TypeSRV   Type = 33

	// Question.Type
	TypeWKS   Type = 11
	TypeHINFO Type = 13
	TypeMINFO Type = 14
	TypeAXFR  Type = 252
	TypeALL   Type = 255

	// ResourceHeader.Class and Question.Class
	ClassINET   Class = 1
	ClassCSNET  Class = 2
	ClassCHAOS  Class = 3
	ClassHESIOD Class = 4

	// Question.Class
	ClassANY Class = 255

	// Message.Rcode
	RCodeSuccess        RCode = 0
	RCodeFormatError    RCode = 1
	RCodeServerFailure  RCode = 2
	RCodeNameError      RCode = 3
	RCodeNotImplemented RCode = 4
	RCodeRefused        RCode = 5
)

var (
	// ErrNotStarted indicates that the prerequisite information isn't
	// available yet because the previous records haven't been appropriately
	// parsed or skipped.
	ErrNotStarted = errors.New("parsing of this type isn't available yet")

	// ErrSectionDone indicated that all records in the section have been
	// parsed.
	ErrSectionDone = errors.New("parsing of this section has completed")

	errBaseLen            = errors.New("insufficient data for base length type")
	errCalcLen            = errors.New("insufficient data for calculated length type")
	errReserved           = errors.New("segment prefix is reserved")
	errTooManyPtr         = errors.New("too many pointers (>10)")
	errInvalidPtr         = errors.New("invalid pointer")
	errResourceLen        = errors.New("insufficient data for resource body length")
	errSegTooLong         = errors.New("segment length too long")
	errZeroSegLen         = errors.New("zero length segment")
	errResTooLong         = errors.New("resource length too long")
	errTooManyQuestions   = errors.New("too many Questions to pack (>65535)")
	errTooManyAnswers     = errors.New("too many Answers to pack (>65535)")
	errTooManyAuthorities = errors.New("too many Authorities to pack (>65535)")
	errTooManyAdditionals = errors.New("too many Additionals to pack (>65535)")
)

type nestedError struct {
	// s is the current level's error message.
	s string

	// err is the nested error.
	err error
}

// nestedError implements error.Error.
func (e *nestedError) Error() string {
	return e.s + ": " + e.err.Error()
}

// Header is a representation of a DNS message header.
type Header struct {
	ID                 uint16
	Response           bool
	OpCode             OpCode
	Authoritative      bool
	Truncated          bool
	RecursionDesired   bool
	RecursionAvailable bool
	RCode              RCode
}

func (m *Header) pack() (id uint16, bits uint16) {
	id = m.ID
	bits = uint16(m.OpCode)<<11 | uint16(m.RCode)
	if m.RecursionAvailable {
		bits |= headerBitRA
	}
	if m.RecursionDesired {
		bits |= headerBitRD
	}
	if m.Truncated {
		bits |= headerBitTC
	}
	if m.Authoritative {
		bits |= headerBitAA
	}
	if m.Response {
		bits |= headerBitQR
	}
	return
}

// Message is a representation of a DNS message.
type Message struct {
	Header
	Questions   []Question
	Answers     []Resource
	Authorities []Resource
	Additionals []Resource
}

type section uint8

const (
	sectionHeader section = iota
	sectionQuestions
	sectionAnswers
	sectionAuthorities
	sectionAdditionals
	sectionDone

	headerBitQR = 1 << 15 // query/response (response=1)
	headerBitAA = 1 << 10 // authoritative
	headerBitTC = 1 << 9  // truncated
	headerBitRD = 1 << 8  // recursion desired
	headerBitRA = 1 << 7  // recursion available
)

var sectionNames = map[section]string{
	sectionHeader:      "header",
	sectionQuestions:   "Question",
	sectionAnswers:     "Answer",
	sectionAuthorities: "Authority",
	sectionAdditionals: "Additional",
}

// header is the wire format for a DNS message header.
type header struct {
	id          uint16
	bits        uint16
	questions   uint16
	answers     uint16
	authorities uint16
	additionals uint16
}

func (h *header) count(sec section) uint16 {
	switch sec {
	case sectionQuestions:
		return h.questions
	case sectionAnswers:
		return h.answers
	case sectionAuthorities:
		return h.authorities
	case sectionAdditionals:
		return h.additionals
	}
	return 0
}

func (h *header) pack(msg []byte) []byte {
	msg = packUint16(msg, h.id)
	msg = packUint16(msg, h.bits)
	msg = packUint16(msg, h.questions)
	msg = packUint16(msg, h.answers)
	msg = packUint16(msg, h.authorities)
	return packUint16(msg, h.additionals)
}

func (h *header) unpack(msg []byte, off int) (int, error) {
	newOff := off
	var err error
	if h.id, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"id", err}
	}
	if h.bits, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"bits", err}
	}
	if h.questions, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"questions", err}
	}
	if h.answers, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"answers", err}
	}
	if h.authorities, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"authorities", err}
	}
	if h.additionals, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"additionals", err}
	}
	return newOff, nil
}

func (h *header) header() Header {
	return Header{
		ID:                 h.id,
		Response:           (h.bits & headerBitQR) != 0,
		OpCode:             OpCode(h.bits>>11) & 0xF,
		Authoritative:      (h.bits & headerBitAA) != 0,
		Truncated:          (h.bits & headerBitTC) != 0,
		RecursionDesired:   (h.bits & headerBitRD) != 0,
		RecursionAvailable: (h.bits & headerBitRA) != 0,
		RCode:              RCode(h.bits & 0xF),
	}
}

// A Resource is a DNS resource record.
type Resource interface {
	// Header return's the Resource's ResourceHeader.
	Header() *ResourceHeader

	// pack packs a Resource except for its header.
	pack(msg []byte, compression map[string]int) ([]byte, error)

	// realType returns the actual type of the Resource. This is used to
	// fill in the header Type field.
	realType() Type
}

func packResource(msg []byte, resource Resource, compression map[string]int) ([]byte, error) {
	oldMsg := msg
	resource.Header().Type = resource.realType()
	msg, length, err := resource.Header().pack(msg, compression)
	if err != nil {
		return msg, &nestedError{"ResourceHeader", err}
	}
	preLen := len(msg)
	msg, err = resource.pack(msg, compression)
	if err != nil {
		return msg, &nestedError{"content", err}
	}
	conLen := len(msg) - preLen
	if conLen > int(^uint16(0)) {
		return oldMsg, errResTooLong
	}
	// Fill in the length now that we know how long the content is.
	packUint16(length[:0], uint16(conLen))
	resource.Header().Length = uint16(conLen)
	return msg, nil
}

// A Parser allows incrementally parsing a DNS message.
//
// When parsing is started, the Header is parsed. Next, each Question can be
// either parsed or skipped. Alternatively, all Questions can be skipped at
// once. When all Questions have been parsed, attempting to parse Questions
// will return (nil, nil) and attempting to skip Questions will return
// (true, nil). After all Questions have been either parsed or skipped, all
// Answers, Authorities and Additionals can be either parsed or skipped in the
// same way, and each type of Resource must be fully parsed or skipped before
// proceeding to the next type of Resource.
//
// Note that there is no requirement to fully skip or parse the message.
type Parser struct {
	msg    []byte
	header header

	section        section
	off            int
	index          int
	resHeaderValid bool
	resHeader      ResourceHeader
}

// Start parses the header and enables the parsing of Questions.
func (p *Parser) Start(msg []byte) (Header, error) {
	if p.msg != nil {
		*p = Parser{}
	}
	p.msg = msg
	var err error
	if p.off, err = p.header.unpack(msg, 0); err != nil {
		return Header{}, &nestedError{"unpacking header", err}
	}
	p.section = sectionQuestions
	return p.header.header(), nil
}

func (p *Parser) checkAdvance(sec section) error {
	if p.section < sec {
		return ErrNotStarted
	}
	if p.section > sec {
		return ErrSectionDone
	}
	p.resHeaderValid = false
	if p.index == int(p.header.count(sec)) {
		p.index = 0
		p.section++
		return ErrSectionDone
	}
	return nil
}

func (p *Parser) resource(sec section) (Resource, error) {
	var r Resource
	hdr, err := p.resourceHeader(sec)
	if err != nil {
		return r, err
	}
	p.resHeaderValid = false
	r, p.off, err = unpackResource(p.msg, p.off, hdr)
	if err != nil {
		return nil, &nestedError{"unpacking " + sectionNames[sec], err}
	}
	p.index++
	return r, nil
}

func (p *Parser) resourceHeader(sec section) (ResourceHeader, error) {
	if p.resHeaderValid {
		return p.resHeader, nil
	}
	if err := p.checkAdvance(sec); err != nil {
		return ResourceHeader{}, err
	}
	var hdr ResourceHeader
	off, err := hdr.unpack(p.msg, p.off)
	if err != nil {
		return ResourceHeader{}, err
	}
	p.resHeaderValid = true
	p.resHeader = hdr
	p.off = off
	return hdr, nil
}

func (p *Parser) skipResource(sec section) error {
	if p.resHeaderValid {
		newOff := p.off + int(p.resHeader.Length)
		if newOff > len(p.msg) {
			return errResourceLen
		}
		p.off = newOff
		p.resHeaderValid = false
		p.index++
		return nil
	}
	if err := p.checkAdvance(sec); err != nil {
		return err
	}
	var err error
	p.off, err = skipResource(p.msg, p.off)
	if err != nil {
		return &nestedError{"skipping: " + sectionNames[sec], err}
	}
	p.index++
	return nil
}

// Question parses a single Question.
func (p *Parser) Question() (Question, error) {
	if err := p.checkAdvance(sectionQuestions); err != nil {
		return Question{}, err
	}
	name, off, err := unpackName(p.msg, p.off)
	if err != nil {
		return Question{}, &nestedError{"unpacking Question.Name", err}
	}
	typ, off, err := unpackType(p.msg, off)
	if err != nil {
		return Question{}, &nestedError{"unpacking Question.Type", err}
	}
	class, off, err := unpackClass(p.msg, off)
	if err != nil {
		return Question{}, &nestedError{"unpacking Question.Class", err}
	}
	p.off = off
	p.index++
	return Question{name, typ, class}, nil
}

// AllQuestions parses all Questions.
func (p *Parser) AllQuestions() ([]Question, error) {
	qs := make([]Question, 0, p.header.questions)
	for {
		q, err := p.Question()
		if err == ErrSectionDone {
			return qs, nil
		}
		if err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
}

// SkipQuestion skips a single Question.
func (p *Parser) SkipQuestion() error {
	if err := p.checkAdvance(sectionQuestions); err != nil {
		return err
	}
	off, err := skipName(p.msg, p.off)
	if err != nil {
		return &nestedError{"skipping Question Name", err}
	}
	if off, err = skipType(p.msg, off); err != nil {
		return &nestedError{"skipping Question Type", err}
	}
	if off, err = skipClass(p.msg, off); err != nil {
		return &nestedError{"skipping Question Class", err}
	}
	p.off = off
	p.index++
	return nil
}

// SkipAllQuestions skips all Questions.
func (p *Parser) SkipAllQuestions() error {
	for {
		if err := p.SkipQuestion(); err == ErrSectionDone {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// AnswerHeader parses a single Answer ResourceHeader.
func (p *Parser) AnswerHeader() (ResourceHeader, error) {
	return p.resourceHeader(sectionAnswers)
}

// Answer parses a single Answer Resource.
func (p *Parser) Answer() (Resource, error) {
	return p.resource(sectionAnswers)
}

// AllAnswers parses all Answer Resources.
func (p *Parser) AllAnswers() ([]Resource, error) {
	as := make([]Resource, 0, p.header.answers)
	for {
		a, err := p.Answer()
		if err == ErrSectionDone {
			return as, nil
		}
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
}

// SkipAnswer skips a single Answer Resource.
func (p *Parser) SkipAnswer() error {
	return p.skipResource(sectionAnswers)
}

// SkipAllAnswers skips all Answer Resources.
func (p *Parser) SkipAllAnswers() error {
	for {
		if err := p.SkipAnswer(); err == ErrSectionDone {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// AuthorityHeader parses a single Authority ResourceHeader.
func (p *Parser) AuthorityHeader() (ResourceHeader, error) {
	return p.resourceHeader(sectionAuthorities)
}

// Authority parses a single Authority Resource.
func (p *Parser) Authority() (Resource, error) {
	return p.resource(sectionAuthorities)
}

// AllAuthorities parses all Authority Resources.
func (p *Parser) AllAuthorities() ([]Resource, error) {
	as := make([]Resource, 0, p.header.authorities)
	for {
		a, err := p.Authority()
		if err == ErrSectionDone {
			return as, nil
		}
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
}

// SkipAuthority skips a single Authority Resource.
func (p *Parser) SkipAuthority() error {
	return p.skipResource(sectionAuthorities)
}

// SkipAllAuthorities skips all Authority Resources.
func (p *Parser) SkipAllAuthorities() error {
	for {
		if err := p.SkipAuthority(); err == ErrSectionDone {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// AdditionalHeader parses a single Additional ResourceHeader.
func (p *Parser) AdditionalHeader() (ResourceHeader, error) {
	return p.resourceHeader(sectionAdditionals)
}

// Additional parses a single Additional Resource.
func (p *Parser) Additional() (Resource, error) {
	return p.resource(sectionAdditionals)
}

// AllAdditionals parses all Additional Resources.
func (p *Parser) AllAdditionals() ([]Resource, error) {
	as := make([]Resource, 0, p.header.additionals)
	for {
		a, err := p.Additional()
		if err == ErrSectionDone {
			return as, nil
		}
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
}

// SkipAdditional skips a single Additional Resource.
func (p *Parser) SkipAdditional() error {
	return p.skipResource(sectionAdditionals)
}

// SkipAllAdditionals skips all Additional Resources.
func (p *Parser) SkipAllAdditionals() error {
	for {
		if err := p.SkipAdditional(); err == ErrSectionDone {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// Unpack parses a full Message.
func (m *Message) Unpack(msg []byte) error {
	var p Parser
	var err error
	if m.Header, err = p.Start(msg); err != nil {
		return err
	}
	if m.Questions, err = p.AllQuestions(); err != nil {
		return err
	}
	if m.Answers, err = p.AllAnswers(); err != nil {
		return err
	}
	if m.Authorities, err = p.AllAuthorities(); err != nil {
		return err
	}
	if m.Additionals, err = p.AllAdditionals(); err != nil {
		return err
	}
	return nil
}

// Pack packs a full Message.
func (m *Message) Pack() ([]byte, error) {
	// Validate the lengths. It is very unlikely that anyone will try to
	// pack more than 65535 of any particular type, but it is possible and
	// we should fail gracefully.
	if len(m.Questions) > int(^uint16(0)) {
		return nil, errTooManyQuestions
	}
	if len(m.Answers) > int(^uint16(0)) {
		return nil, errTooManyAnswers
	}
	if len(m.Authorities) > int(^uint16(0)) {
		return nil, errTooManyAuthorities
	}
	if len(m.Additionals) > int(^uint16(0)) {
		return nil, errTooManyAdditionals
	}

	var h header
	h.id, h.bits = m.Header.pack()

	h.questions = uint16(len(m.Questions))
	h.answers = uint16(len(m.Answers))
	h.authorities = uint16(len(m.Authorities))
	h.additionals = uint16(len(m.Additionals))

	// The starting capacity doesn't matter too much, but most DNS responses
	// Will be <= 512 bytes as it is the limit for DNS over UDP.
	msg := make([]byte, 0, 512)

	msg = h.pack(msg)

	// RFC 1035 allows (but does not require) compression for packing. RFC
	// 1035 requires unpacking implementations to support compression, so
	// unconditionally enabling it is fine.
	//
	// DNS lookups are typically done over UDP, and RFC 1035 states that UDP
	// DNS packets can be a maximum of 512 bytes long. Without compression,
	// many DNS response packets are over this limit, so enabling
	// compression will help ensure compliance.
	compression := map[string]int{}

	for _, q := range m.Questions {
		var err error
		msg, err = q.pack(msg, compression)
		if err != nil {
			return nil, &nestedError{"packing Question", err}
		}
	}
	for _, a := range m.Answers {
		var err error
		msg, err = packResource(msg, a, compression)
		if err != nil {
			return nil, &nestedError{"packing Answer", err}
		}
	}
	for _, a := range m.Authorities {
		var err error
		msg, err = packResource(msg, a, compression)
		if err != nil {
			return nil, &nestedError{"packing Authority", err}
		}
	}
	for _, a := range m.Additionals {
		var err error
		msg, err = packResource(msg, a, compression)
		if err != nil {
			return nil, &nestedError{"packing Additional", err}
		}
	}

	return msg, nil
}

// An ResourceHeader is the header of a DNS resource record. There are
// many types of DNS resource records, but they all share the same header.
type ResourceHeader struct {
	// Name is the domain name for which this resource record pertains.
	Name string

	// Type is the type of DNS resource record.
	//
	// This field will be set automatically during packing.
	Type Type

	// Class is the class of network to which this DNS resource record
	// pertains.
	Class Class

	// TTL is the length of time (measured in seconds) which this resource
	// record is valid for (time to live). All Resources in a set should
	// have the same TTL (RFC 2181 Section 5.2).
	TTL uint32

	// Length is the length of data in the resource record after the header.
	//
	// This field will be set automatically during packing.
	Length uint16
}

// Header implements Resource.Header.
func (h *ResourceHeader) Header() *ResourceHeader {
	return h
}

// pack packs all of the fields in a ResourceHeader except for the length. The
// length bytes are returned as a slice so they can be filled in after the rest
// of the Resource has been packed.
func (h *ResourceHeader) pack(oldMsg []byte, compression map[string]int) (msg []byte, length []byte, err error) {
	msg = oldMsg
	if msg, err = packName(msg, h.Name, compression); err != nil {
		return oldMsg, nil, &nestedError{"Name", err}
	}
	msg = packType(msg, h.Type)
	msg = packClass(msg, h.Class)
	msg = packUint32(msg, h.TTL)
	lenBegin := len(msg)
	msg = packUint16(msg, h.Length)
	return msg, msg[lenBegin:], nil
}

func (h *ResourceHeader) unpack(msg []byte, off int) (int, error) {
	newOff := off
	var err error
	if h.Name, newOff, err = unpackName(msg, newOff); err != nil {
		return off, &nestedError{"Name", err}
	}
	if h.Type, newOff, err = unpackType(msg, newOff); err != nil {
		return off, &nestedError{"Type", err}
	}
	if h.Class, newOff, err = unpackClass(msg, newOff); err != nil {
		return off, &nestedError{"Class", err}
	}
	if h.TTL, newOff, err = unpackUint32(msg, newOff); err != nil {
		return off, &nestedError{"TTL", err}
	}
	if h.Length, newOff, err = unpackUint16(msg, newOff); err != nil {
		return off, &nestedError{"Length", err}
	}
	return newOff, nil
}

func skipResource(msg []byte, off int) (int, error) {
	newOff, err := skipName(msg, off)
	if err != nil {
		return off, &nestedError{"Name", err}
	}
	if newOff, err = skipType(msg, newOff); err != nil {
		return off, &nestedError{"Type", err}
	}
	if newOff, err = skipClass(msg, newOff); err != nil {
		return off, &nestedError{"Class", err}
	}
	if newOff, err = skipUint32(msg, newOff); err != nil {
		return off, &nestedError{"TTL", err}
	}
	length, newOff, err := unpackUint16(msg, newOff)
	if err != nil {
		return off, &nestedError{"Length", err}
	}
	if newOff += int(length); newOff > len(msg) {
		return off, errResourceLen
	}
	return newOff, nil
}

func packUint16(msg []byte, field uint16) []byte {
	return append(msg, byte(field>>8), byte(field))
}

func unpackUint16(msg []byte, off int) (uint16, int, error) {
	if off+2 > len(msg) {
		return 0, off, errBaseLen
	}
	return uint16(msg[off])<<8 | uint16(msg[off+1]), off + 2, nil
}

func skipUint16(msg []byte, off int) (int, error) {
	if off+2 > len(msg) {
		return off, errBaseLen
	}
	return off + 2, nil
}

func packType(msg []byte, field Type) []byte {
	return packUint16(msg, uint16(field))
}

func unpackType(msg []byte, off int) (Type, int, error) {
	t, o, err := unpackUint16(msg, off)
	return Type(t), o, err
}

func skipType(msg []byte, off int) (int, error) {
	return skipUint16(msg, off)
}

func packClass(msg []byte, field Class) []byte {
	return packUint16(msg, uint16(field))
}

func unpackClass(msg []byte, off int) (Class, int, error) {
	c, o, err := unpackUint16(msg, off)
	return Class(c), o, err
}

func skipClass(msg []byte, off int) (int, error) {
	return skipUint16(msg, off)
}

func packUint32(msg []byte, field uint32) []byte {
	return append(
		msg,
		byte(field>>24),
		byte(field>>16),
		byte(field>>8),
		byte(field),
	)
}

func unpackUint32(msg []byte, off int) (uint32, int, error) {
	if off+4 > len(msg) {
		return 0, off, errBaseLen
	}
	v := uint32(msg[off])<<24 | uint32(msg[off+1])<<16 | uint32(msg[off+2])<<8 | uint32(msg[off+3])
	return v, off + 4, nil
}

func skipUint32(msg []byte, off int) (int, error) {
	if off+4 > len(msg) {
		return off, errBaseLen
	}
	return off + 4, nil
}

func packText(msg []byte, field string) []byte {
	for len(field) > 0 {
		l := len(field)
		if l > 255 {
			l = 255
		}
		msg = append(msg, byte(l))
		msg = append(msg, field[:l]...)
		field = field[l:]
	}
	return msg
}

func unpackText(msg []byte, off int) (string, int, error) {
	if off >= len(msg) {
		return "", off, errBaseLen
	}
	beginOff := off + 1
	endOff := beginOff + int(msg[off])
	if endOff > len(msg) {
		return "", off, errCalcLen
	}
	return string(msg[beginOff:endOff]), endOff, nil
}

func skipText(msg []byte, off int) (int, error) {
	if off >= len(msg) {
		return off, errBaseLen
	}
	endOff := off + 1 + int(msg[off])
	if endOff > len(msg) {
		return off, errCalcLen
	}
	return endOff, nil
}

func packBytes(msg []byte, field []byte) []byte {
	return append(msg, field...)
}

func unpackBytes(msg []byte, off int, field []byte) (int, error) {
	newOff := off + len(field)
	if newOff > len(msg) {
		return off, errBaseLen
	}
	copy(field, msg[off:newOff])
	return newOff, nil
}

func skipBytes(msg []byte, off int, field []byte) (int, error) {
	newOff := off + len(field)
	if newOff > len(msg) {
		return off, errBaseLen
	}
	return newOff, nil
}

// packName packs a domain name.
//
// Domain names are a sequence of counted strings split at the dots. They end
// with a zero-length string. Compression can be used to reuse domain suffixes.
//
// The compression map will be updated with new domain suffixes. If compression
// is nil, compression will not be used.
func packName(msg []byte, name string, compression map[string]int) ([]byte, error) {
	oldMsg := msg

	// Add a trailing dot to canonicalize name.
	if n := len(name); n == 0 || name[n-1] != '.' {
		name += "."
	}

	// Allow root domain.
	if name == "." {
		return append(msg, 0), nil
	}

	// Emit sequence of counted strings, chopping at dots.
	for i, begin := 0, 0; i < len(name); i++ {
		// Check for the end of the segment.
		if name[i] == '.' {
			// The two most significant bits have special meaning.
			// It isn't allowed for segments to be long enough to
			// need them.
			if i-begin >= 1<<6 {
				return oldMsg, errSegTooLong
			}

			// Segments must have a non-zero length.
			if i-begin == 0 {
				return oldMsg, errZeroSegLen
			}

			msg = append(msg, byte(i-begin))

			for j := begin; j < i; j++ {
				msg = append(msg, name[j])
			}

			begin = i + 1
			continue
		}

		// We can only compress domain suffixes starting with a new
		// segment. A pointer is two bytes with the two most significant
		// bits set to 1 to indicate that it is a pointer.
		if (i == 0 || name[i-1] == '.') && compression != nil {
			if ptr, ok := compression[name[i:]]; ok {
				// Hit. Emit a pointer instead of the rest of
				// the domain.
				return append(msg, byte(ptr>>8|0xC0), byte(ptr)), nil
			}

			// Miss. Add the suffix to the compression table if the
			// offset can be stored in the available 14 bytes.
			if len(msg) <= int(^uint16(0)>>2) {
				compression[name[i:]] = len(msg)
			}
		}
	}
	return append(msg, 0), nil
}

// unpackName unpacks a domain name.
func unpackName(msg []byte, off int) (string, int, error) {
	// currOff is the current working offset.
	currOff := off

	// newOff is the offset where the next record will start. Pointers lead
	// to data that belongs to other names and thus doesn't count towards to
	// the usage of this name.
	newOff := off

	// name is the domain name being unpacked.
	name := make([]byte, 0, 255)

	// ptr is the number of pointers followed.
	var ptr int
Loop:
	for {
		if currOff >= len(msg) {
			return "", off, errBaseLen
		}
		c := int(msg[currOff])
		currOff++
		switch c & 0xC0 {
		case 0x00: // String segment
			if c == 0x00 {
				// A zero length signals the end of the name.
				break Loop
			}
			endOff := currOff + c
			if endOff > len(msg) {
				return "", off, errCalcLen
			}
			name = append(name, msg[currOff:endOff]...)
			name = append(name, '.')
			currOff = endOff
		case 0xC0: // Pointer
			if currOff >= len(msg) {
				return "", off, errInvalidPtr
			}
			c1 := msg[currOff]
			currOff++
			if ptr == 0 {
				newOff = currOff
			}
			// Don't follow too many pointers, maybe there's a loop.
			if ptr++; ptr > 10 {
				return "", off, errTooManyPtr
			}
			currOff = (c^0xC0)<<8 | int(c1)
		default:
			// Prefixes 0x80 and 0x40 are reserved.
			return "", off, errReserved
		}
	}
	if len(name) == 0 {
		name = append(name, '.')
	}
	if ptr == 0 {
		newOff = currOff
	}
	return string(name), newOff, nil
}

func skipName(msg []byte, off int) (int, error) {
	// newOff is the offset where the next record will start. Pointers lead
	// to data that belongs to other names and thus doesn't count towards to
	// the usage of this name.
	newOff := off

Loop:
	for {
		if newOff >= len(msg) {
			return off, errBaseLen
		}
		c := int(msg[newOff])
		newOff++
		switch c & 0xC0 {
		case 0x00:
			if c == 0x00 {
				// A zero length signals the end of the name.
				break Loop
			}
			// literal string
			newOff += c
			if newOff > len(msg) {
				return off, errCalcLen
			}
		case 0xC0:
			// Pointer to somewhere else in msg.

			// Pointers are two bytes.
			newOff++

			// Don't follow the pointer as the data here has ended.
			break Loop
		default:
			// Prefixes 0x80 and 0x40 are reserved.
			return off, errReserved
		}
	}

	return newOff, nil
}

// A Question is a DNS query.
type Question struct {
	Name  string
	Type  Type
	Class Class
}

func (q *Question) pack(msg []byte, compression map[string]int) ([]byte, error) {
	msg, err := packName(msg, q.Name, compression)
	if err != nil {
		return msg, &nestedError{"Name", err}
	}
	msg = packType(msg, q.Type)
	return packClass(msg, q.Class), nil
}

func unpackResource(msg []byte, off int, hdr ResourceHeader) (Resource, int, error) {
	var (
		r    Resource
		err  error
		name string
	)
	switch hdr.Type {
	case TypeA:
		r, err = unpackAResource(hdr, msg, off)
		name = "A"
	case TypeNS:
		r, err = unpackNSResource(hdr, msg, off)
		name = "NS"
	case TypeCNAME:
		r, err = unpackCNAMEResource(hdr, msg, off)
		name = "CNAME"
	case TypeSOA:
		r, err = unpackSOAResource(hdr, msg, off)
		name = "SOA"
	case TypePTR:
		r, err = unpackPTRResource(hdr, msg, off)
		name = "PTR"
	case TypeMX:
		r, err = unpackMXResource(hdr, msg, off)
		name = "MX"
	case TypeTXT:
		r, err = unpackTXTResource(hdr, msg, off)
		name = "TXT"
	case TypeAAAA:
		r, err = unpackAAAAResource(hdr, msg, off)
		name = "AAAA"
	case TypeSRV:
		r, err = unpackSRVResource(hdr, msg, off)
		name = "SRV"
	}
	if err != nil {
		return nil, off, &nestedError{name + " record", err}
	}
	if r != nil {
		return r, off + int(hdr.Length), nil
	}
	return nil, off, errors.New("invalid resource type: " + string(hdr.Type+'0'))
}

// A CNAMEResource is a CNAME Resource record.
type CNAMEResource struct {
	ResourceHeader

	CNAME string
}

func (r *CNAMEResource) realType() Type {
	return TypeCNAME
}

func (r *CNAMEResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packName(msg, r.CNAME, compression)
}

func unpackCNAMEResource(hdr ResourceHeader, msg []byte, off int) (*CNAMEResource, error) {
	cname, _, err := unpackName(msg, off)
	if err != nil {
		return nil, err
	}
	return &CNAMEResource{hdr, cname}, nil
}

// An MXResource is an MX Resource record.
type MXResource struct {
	ResourceHeader

	Pref uint16
	MX   string
}

func (r *MXResource) realType() Type {
	return TypeMX
}

func (r *MXResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	oldMsg := msg
	msg = packUint16(msg, r.Pref)
	msg, err := packName(msg, r.MX, compression)
	if err != nil {
		return oldMsg, &nestedError{"MXResource.MX", err}
	}
	return msg, nil
}

func unpackMXResource(hdr ResourceHeader, msg []byte, off int) (*MXResource, error) {
	pref, off, err := unpackUint16(msg, off)
	if err != nil {
		return nil, &nestedError{"Pref", err}
	}
	mx, _, err := unpackName(msg, off)
	if err != nil {
		return nil, &nestedError{"MX", err}
	}
	return &MXResource{hdr, pref, mx}, nil
}

// An NSResource is an NS Resource record.
type NSResource struct {
	ResourceHeader

	NS string
}

func (r *NSResource) realType() Type {
	return TypeNS
}

func (r *NSResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packName(msg, r.NS, compression)
}

func unpackNSResource(hdr ResourceHeader, msg []byte, off int) (*NSResource, error) {
	ns, _, err := unpackName(msg, off)
	if err != nil {
		return nil, err
	}
	return &NSResource{hdr, ns}, nil
}

// A PTRResource is a PTR Resource record.
type PTRResource struct {
	ResourceHeader

	PTR string
}

func (r *PTRResource) realType() Type {
	return TypePTR
}

func (r *PTRResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packName(msg, r.PTR, compression)
}

func unpackPTRResource(hdr ResourceHeader, msg []byte, off int) (*PTRResource, error) {
	ptr, _, err := unpackName(msg, off)
	if err != nil {
		return nil, err
	}
	return &PTRResource{hdr, ptr}, nil
}

// An SOAResource is an SOA Resource record.
type SOAResource struct {
	ResourceHeader

	NS      string
	MBox    string
	Serial  uint32
	Refresh uint32
	Retry   uint32
	Expire  uint32

	// MinTTL the is the default TTL of Resources records which did not
	// contain a TTL value and the TTL of negative responses. (RFC 2308
	// Section 4)
	MinTTL uint32
}

func (r *SOAResource) realType() Type {
	return TypeSOA
}

func (r *SOAResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	oldMsg := msg
	msg, err := packName(msg, r.NS, compression)
	if err != nil {
		return oldMsg, &nestedError{"SOAResource.NS", err}
	}
	msg, err = packName(msg, r.MBox, compression)
	if err != nil {
		return oldMsg, &nestedError{"SOAResource.MBox", err}
	}
	msg = packUint32(msg, r.Serial)
	msg = packUint32(msg, r.Refresh)
	msg = packUint32(msg, r.Retry)
	msg = packUint32(msg, r.Expire)
	return packUint32(msg, r.MinTTL), nil
}

func unpackSOAResource(hdr ResourceHeader, msg []byte, off int) (*SOAResource, error) {
	ns, off, err := unpackName(msg, off)
	if err != nil {
		return nil, &nestedError{"NS", err}
	}
	mbox, off, err := unpackName(msg, off)
	if err != nil {
		return nil, &nestedError{"MBox", err}
	}
	serial, off, err := unpackUint32(msg, off)
	if err != nil {
		return nil, &nestedError{"Serial", err}
	}
	refresh, off, err := unpackUint32(msg, off)
	if err != nil {
		return nil, &nestedError{"Refresh", err}
	}
	retry, off, err := unpackUint32(msg, off)
	if err != nil {
		return nil, &nestedError{"Retry", err}
	}
	expire, off, err := unpackUint32(msg, off)
	if err != nil {
		return nil, &nestedError{"Expire", err}
	}
	minTTL, _, err := unpackUint32(msg, off)
	if err != nil {
		return nil, &nestedError{"MinTTL", err}
	}
	return &SOAResource{hdr, ns, mbox, serial, refresh, retry, expire, minTTL}, nil
}

// A TXTResource is a TXT Resource record.
type TXTResource struct {
	ResourceHeader

	Txt string // Not a domain name.
}

func (r *TXTResource) realType() Type {
	return TypeTXT
}

func (r *TXTResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packText(msg, r.Txt), nil
}

func unpackTXTResource(hdr ResourceHeader, msg []byte, off int) (*TXTResource, error) {
	var txt string
	for n := uint16(0); n < hdr.Length; {
		var t string
		var err error
		if t, off, err = unpackText(msg, off); err != nil {
			return nil, &nestedError{"text", err}
		}
		// Check if we got too many bytes.
		if hdr.Length-n < uint16(len(t))+1 {
			return nil, errCalcLen
		}
		n += uint16(len(t)) + 1
		txt += t
	}
	return &TXTResource{hdr, txt}, nil
}

// An SRVResource is an SRV Resource record.
type SRVResource struct {
	ResourceHeader

	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string // Not compressed as per RFC 2782.
}

func (r *SRVResource) realType() Type {
	return TypeSRV
}

func (r *SRVResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	oldMsg := msg
	msg = packUint16(msg, r.Priority)
	msg = packUint16(msg, r.Weight)
	msg = packUint16(msg, r.Port)
	msg, err := packName(msg, r.Target, nil)
	if err != nil {
		return oldMsg, &nestedError{"SRVResource.Target", err}
	}
	return msg, nil
}

func unpackSRVResource(hdr ResourceHeader, msg []byte, off int) (*SRVResource, error) {
	priority, off, err := unpackUint16(msg, off)
	if err != nil {
		return nil, &nestedError{"Priority", err}
	}
	weight, off, err := unpackUint16(msg, off)
	if err != nil {
		return nil, &nestedError{"Weight", err}
	}
	port, off, err := unpackUint16(msg, off)
	if err != nil {
		return nil, &nestedError{"Port", err}
	}
	target, _, err := unpackName(msg, off)
	if err != nil {
		return nil, &nestedError{"Target", err}
	}
	return &SRVResource{hdr, priority, weight, port, target}, nil
}

// An AResource is an A Resource record.
type AResource struct {
	ResourceHeader

	A [4]byte
}

func (r *AResource) realType() Type {
	return TypeA
}

func (r *AResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packBytes(msg, r.A[:]), nil
}

func unpackAResource(hdr ResourceHeader, msg []byte, off int) (*AResource, error) {
	var a [4]byte
	if _, err := unpackBytes(msg, off, a[:]); err != nil {
		return nil, err
	}
	return &AResource{hdr, a}, nil
}

// An AAAAResource is an AAAA Resource record.
type AAAAResource struct {
	ResourceHeader

	AAAA [16]byte
}

func (r *AAAAResource) realType() Type {
	return TypeAAAA
}

func (r *AAAAResource) pack(msg []byte, compression map[string]int) ([]byte, error) {
	return packBytes(msg, r.AAAA[:]), nil
}

func unpackAAAAResource(hdr ResourceHeader, msg []byte, off int) (*AAAAResource, error) {
	var aaaa [16]byte
	if _, err := unpackBytes(msg, off, aaaa[:]); err != nil {
		return nil, err
	}
	return &AAAAResource{hdr, aaaa}, nil
}
