// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dnsmessage

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
)

func (m *Message) String() string {
	s := fmt.Sprintf("Message: %#v\n", &m.Header)
	if len(m.Questions) > 0 {
		s += "-- Questions\n"
		for _, q := range m.Questions {
			s += fmt.Sprintf("%#v\n", q)
		}
	}
	if len(m.Answers) > 0 {
		s += "-- Answers\n"
		for _, a := range m.Answers {
			s += fmt.Sprintf("%#v\n", a)
		}
	}
	if len(m.Authorities) > 0 {
		s += "-- Authorities\n"
		for _, ns := range m.Authorities {
			s += fmt.Sprintf("%#v\n", ns)
		}
	}
	if len(m.Additionals) > 0 {
		s += "-- Additionals\n"
		for _, e := range m.Additionals {
			s += fmt.Sprintf("%#v\n", e)
		}
	}
	return s
}

func TestQuestionPackUnpack(t *testing.T) {
	want := Question{
		Name:  ".",
		Type:  TypeA,
		Class: ClassINET,
	}
	buf, err := want.pack(make([]byte, 1, 50), map[string]int{})
	if err != nil {
		t.Fatal("Packing failed:", err)
	}
	var p Parser
	p.msg = buf
	p.header.questions = 1
	p.section = sectionQuestions
	p.off = 1
	got, err := p.Question()
	if err != nil {
		t.Fatalf("Unpacking failed: %v\n%s", err, string(buf[1:]))
	}
	if p.off != len(buf) {
		t.Errorf("Unpacked different amount than packed: got n = %d, want = %d", p.off, len(buf))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got = %+v, want = %+v", got, want)
	}
}

func TestNamePackUnpack(t *testing.T) {
	tests := []struct {
		in   string
		want string
		err  error
	}{
		{"", ".", nil},
		{".", ".", nil},
		{"google..com", "", errZeroSegLen},
		{"google.com", "google.com.", nil},
		{"google..com.", "", errZeroSegLen},
		{"google.com.", "google.com.", nil},
		{".google.com.", "", errZeroSegLen},
		{"www..google.com.", "", errZeroSegLen},
		{"www.google.com.", "www.google.com.", nil},
	}

	for _, test := range tests {
		buf, err := packName(make([]byte, 0, 30), test.in, map[string]int{})
		if err != test.err {
			t.Errorf("Packing of %s: got err = %v, want err = %v", test.in, err, test.err)
			continue
		}
		if test.err != nil {
			continue
		}
		got, n, err := unpackName(buf, 0)
		if err != nil {
			t.Errorf("Unpacking for %s failed: %v", test.in, err)
			continue
		}
		if n != len(buf) {
			t.Errorf(
				"Unpacked different amount than packed for %s: got n = %d, want = %d",
				test.in,
				n,
				len(buf),
			)
		}
		if got != test.want {
			t.Errorf("Unpacking packing of %s: got = %s, want = %s", test.in, got, test.want)
		}
	}
}

func TestDNSPackUnpack(t *testing.T) {
	wants := []Message{
		{
			Questions: []Question{
				{
					Name:  ".",
					Type:  TypeAAAA,
					Class: ClassINET,
				},
			},
			Answers:     []Resource{},
			Authorities: []Resource{},
			Additionals: []Resource{},
		},
		largeTestMsg(),
	}
	for i, want := range wants {
		b, err := want.Pack()
		if err != nil {
			t.Fatalf("%d: packing failed: %v", i, err)
		}
		var got Message
		err = got.Unpack(b)
		if err != nil {
			t.Fatalf("%d: unpacking failed: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got = %+v, want = %+v", i, &got, &want)
		}
	}
}

func TestSkipAll(t *testing.T) {
	msg := largeTestMsg()
	buf, err := msg.Pack()
	if err != nil {
		t.Fatal("Packing large test message:", err)
	}
	var p Parser
	if _, err := p.Start(buf); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		f    func() error
	}{
		{"SkipAllQuestions", p.SkipAllQuestions},
		{"SkipAllAnswers", p.SkipAllAnswers},
		{"SkipAllAuthorities", p.SkipAllAuthorities},
		{"SkipAllAdditionals", p.SkipAllAdditionals},
	}
	for _, test := range tests {
		for i := 1; i <= 3; i++ {
			if err := test.f(); err != nil {
				t.Errorf("Call #%d to %s(): %v", i, test.name, err)
			}
		}
	}
}

func TestSkipNotStarted(t *testing.T) {
	var p Parser

	tests := []struct {
		name string
		f    func() error
	}{
		{"SkipAllQuestions", p.SkipAllQuestions},
		{"SkipAllAnswers", p.SkipAllAnswers},
		{"SkipAllAuthorities", p.SkipAllAuthorities},
		{"SkipAllAdditionals", p.SkipAllAdditionals},
	}
	for _, test := range tests {
		if err := test.f(); err != ErrNotStarted {
			t.Errorf("Got %s() = %v, want = %v", test.name, err, ErrNotStarted)
		}
	}
}

func TestTooManyRecords(t *testing.T) {
	const recs = int(^uint16(0)) + 1
	tests := []struct {
		name string
		msg  Message
		want error
	}{
		{
			"Questions",
			Message{
				Questions: make([]Question, recs),
			},
			errTooManyQuestions,
		},
		{
			"Answers",
			Message{
				Answers: make([]Resource, recs),
			},
			errTooManyAnswers,
		},
		{
			"Authorities",
			Message{
				Authorities: make([]Resource, recs),
			},
			errTooManyAuthorities,
		},
		{
			"Additionals",
			Message{
				Additionals: make([]Resource, recs),
			},
			errTooManyAdditionals,
		},
	}

	for _, test := range tests {
		if _, got := test.msg.Pack(); got != test.want {
			t.Errorf("Packing %d %s: got = %v, want = %v", recs, test.name, got, test.want)
		}
	}
}

func TestVeryLongTxt(t *testing.T) {
	want := &TXTResource{
		ResourceHeader: ResourceHeader{
			Name:  "foo.bar.example.com.",
			Type:  TypeTXT,
			Class: ClassINET,
		},
		Txt: loremIpsum,
	}
	buf, err := packResource(make([]byte, 0, 8000), want, map[string]int{})
	if err != nil {
		t.Fatal("Packing failed:", err)
	}
	var hdr ResourceHeader
	off, err := hdr.unpack(buf, 0)
	if err != nil {
		t.Fatal("Unpacking ResourceHeader failed:", err)
	}
	got, n, err := unpackResource(buf, off, hdr)
	if err != nil {
		t.Fatal("Unpacking failed:", err)
	}
	if n != len(buf) {
		t.Errorf("Unpacked different amount than packed: got n = %d, want = %d", n, len(buf))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got = %+v, want = %+v", got, want)
	}
}

func ExampleHeaderSearch() {
	msg := Message{
		Header: Header{Response: true, Authoritative: true},
		Questions: []Question{
			{
				Name:  "foo.bar.example.com.",
				Type:  TypeA,
				Class: ClassINET,
			},
			{
				Name:  "bar.example.com.",
				Type:  TypeA,
				Class: ClassINET,
			},
		},
		Answers: []Resource{
			&AResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeA,
					Class: ClassINET,
				},
				A: [4]byte{127, 0, 0, 1},
			},
			&AResource{
				ResourceHeader: ResourceHeader{
					Name:  "bar.example.com.",
					Type:  TypeA,
					Class: ClassINET,
				},
				A: [4]byte{127, 0, 0, 2},
			},
		},
	}

	buf, err := msg.Pack()
	if err != nil {
		panic(err)
	}

	wantName := "bar.example.com."

	var p Parser
	if _, err := p.Start(buf); err != nil {
		panic(err)
	}

	for {
		q, err := p.Question()
		if err == ErrSectionDone {
			break
		}
		if err != nil {
			panic(err)
		}

		if q.Name != wantName {
			continue
		}

		fmt.Println("Found question for name", wantName)
		if err := p.SkipAllQuestions(); err != nil {
			panic(err)
		}
		break
	}

	var gotIPs []net.IP
	for {
		h, err := p.AnswerHeader()
		if err == ErrSectionDone {
			break
		}
		if err != nil {
			panic(err)
		}

		if (h.Type != TypeA && h.Type != TypeAAAA) || h.Class != ClassINET {
			continue
		}

		if !strings.EqualFold(h.Name, wantName) {
			if err := p.SkipAnswer(); err != nil {
				panic(err)
			}
			continue
		}
		a, err := p.Answer()
		if err != nil {
			panic(err)
		}

		switch r := a.(type) {
		default:
			panic(fmt.Sprintf("unknown type: %T", r))
		case *AResource:
			gotIPs = append(gotIPs, r.A[:])
		case *AAAAResource:
			gotIPs = append(gotIPs, r.AAAA[:])
		}
	}

	fmt.Printf("Found A/AAAA records for name %s: %v\n", wantName, gotIPs)

	// Output:
	// Found question for name bar.example.com.
	// Found A/AAAA records for name bar.example.com.: [127.0.0.2]
}

func largeTestMsg() Message {
	return Message{
		Header: Header{Response: true, Authoritative: true},
		Questions: []Question{
			{
				Name:  "foo.bar.example.com.",
				Type:  TypeA,
				Class: ClassINET,
			},
		},
		Answers: []Resource{
			&AResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeA,
					Class: ClassINET,
				},
				A: [4]byte{127, 0, 0, 1},
			},
			&AResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeA,
					Class: ClassINET,
				},
				A: [4]byte{127, 0, 0, 2},
			},
		},
		Authorities: []Resource{
			&NSResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeNS,
					Class: ClassINET,
				},
				NS: "ns1.example.com.",
			},
			&NSResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeNS,
					Class: ClassINET,
				},
				NS: "ns2.example.com.",
			},
		},
		Additionals: []Resource{
			&TXTResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeTXT,
					Class: ClassINET,
				},
				Txt: "So Long, and Thanks for All the Fish",
			},
			&TXTResource{
				ResourceHeader: ResourceHeader{
					Name:  "foo.bar.example.com.",
					Type:  TypeTXT,
					Class: ClassINET,
				},
				Txt: "Hamster Huey and the Gooey Kablooie",
			},
		},
	}
}

const loremIpsum = `
Lorem ipsum dolor sit amet, nec enim antiopam id, an ullum choro
nonumes qui, pro eu debet honestatis mediocritatem. No alia enim eos,
magna signiferumque ex vis. Mei no aperiri dissentias, cu vel quas
regione. Malorum quaeque vim ut, eum cu semper aliquid invidunt, ei
nam ipsum assentior.

Nostrum appellantur usu no, vis ex probatus adipiscing. Cu usu illum
facilis eleifend. Iusto conceptam complectitur vim id. Tale omnesque
no usu, ei oblique sadipscing vim. At nullam voluptua usu, mei laudem
reformidans et. Qui ei eros porro reformidans, ius suas veritus
torquatos ex. Mea te facer alterum consequat.

Soleat torquatos democritum sed et, no mea congue appareat, facer
aliquam nec in. Has te ipsum tritani. At justo dicta option nec, movet
phaedrum ad nam. Ea detracto verterem liberavisse has, delectus
suscipiantur in mei. Ex nam meliore complectitur. Ut nam omnis
honestatis quaerendum, ea mea nihil affert detracto, ad vix rebum
mollis.

Ut epicurei praesent neglegentur pri, prima fuisset intellegebat ad
vim. An habemus comprehensam usu, at enim dignissim pro. Eam reque
vivendum adipisci ea. Vel ne odio choro minimum. Sea admodum
dissentiet ex. Mundi tamquam evertitur ius cu. Homero postea iisque ut
pro, vel ne saepe senserit consetetur.

Nulla utamur facilisis ius ea, in viderer diceret pertinax eum. Mei no
enim quodsi facilisi, ex sed aeterno appareat mediocritatem, eum
sententiae deterruisset ut. At suas timeam euismod cum, offendit
appareat interpretaris ne vix. Vel ea civibus albucius, ex vim quidam
accusata intellegebat, noluisse instructior sea id. Nec te nonumes
habemus appellantur, quis dignissim vituperata eu nam.

At vix apeirian patrioque vituperatoribus, an usu agam assum. Debet
iisque an mea. Per eu dicant ponderum accommodare. Pri alienum
placerat senserit an, ne eum ferri abhorreant vituperatoribus. Ut mea
eligendi disputationi. Ius no tation everti impedit, ei magna quidam
mediocritatem pri.

Legendos perpetua iracundia ne usu, no ius ullum epicurei intellegam,
ad modus epicuri lucilius eam. In unum quaerendum usu. Ne diam paulo
has, ea veri virtute sed. Alia honestatis conclusionemque mea eu, ut
iudico albucius his.

Usu essent probatus eu, sed omnis dolor delicatissimi ex. No qui augue
dissentias dissentiet. Laudem recteque no usu, vel an velit noluisse,
an sed utinam eirmod appetere. Ne mea fuisset inimicus ocurreret. At
vis dicant abhorreant, utinam forensibus nec ne, mei te docendi
consequat. Brute inermis persecuti cum id. Ut ipsum munere propriae
usu, dicit graeco disputando id has.

Eros dolore quaerendum nam ei. Timeam ornatus inciderint pro id. Nec
torquatos sadipscing ei, ancillae molestie per in. Malis principes duo
ea, usu liber postulant ei.

Graece timeam voluptatibus eu eam. Alia probatus quo no, ea scripta
feugiat duo. Congue option meliore ex qui, noster invenire appellantur
ea vel. Eu exerci legendos vel. Consetetur repudiandae vim ut. Vix an
probo minimum, et nam illud falli tempor.

Cum dico signiferumque eu. Sed ut regione maiorum, id veritus insolens
tacimates vix. Eu mel sint tamquam lucilius, duo no oporteat
tacimates. Atqui augue concludaturque vix ei, id mel utroque menandri.

Ad oratio blandit aliquando pro. Vis et dolorum rationibus
philosophia, ad cum nulla molestie. Hinc fuisset adversarium eum et,
ne qui nisl verear saperet, vel te quaestio forensibus. Per odio
option delenit an. Alii placerat has no, in pri nihil platonem
cotidieque. Est ut elit copiosae scaevola, debet tollit maluisset sea
an.

Te sea hinc debet pericula, liber ridens fabulas cu sed, quem mutat
accusam mea et. Elitr labitur albucius et pri, an labore feugait mel.
Velit zril melius usu ea. Ad stet putent interpretaris qui. Mel no
error volumus scripserit. In pro paulo iudico, quo ei dolorem
verterem, affert fabellas dissentiet ea vix.

Vis quot deserunt te. Error aliquid detraxit eu usu, vis alia eruditi
salutatus cu. Est nostrud bonorum an, ei usu alii salutatus. Vel at
nisl primis, eum ex aperiri noluisse reformidans. Ad veri velit
utroque vis, ex equidem detraxit temporibus has.

Inermis appareat usu ne. Eros placerat periculis mea ad, in dictas
pericula pro. Errem postulant at usu, ea nec amet ornatus mentitum. Ad
mazim graeco eum, vel ex percipit volutpat iudicabit, sit ne delicata
interesset. Mel sapientem prodesset abhorreant et, oblique suscipit
eam id.

An maluisset disputando mea, vidit mnesarchum pri et. Malis insolens
inciderint no sea. Ea persius maluisset vix, ne vim appellantur
instructior, consul quidam definiebas pri id. Cum integre feugiat
pericula in, ex sed persius similique, mel ne natum dicit percipitur.

Primis discere ne pri, errem putent definitionem at vis. Ei mel dolore
neglegentur, mei tincidunt percipitur ei. Pro ad simul integre
rationibus. Eu vel alii honestatis definitiones, mea no nonumy
reprehendunt.

Dicta appareat legendos est cu. Eu vel congue dicunt omittam, no vix
adhuc minimum constituam, quot noluisse id mel. Eu quot sale mutat
duo, ex nisl munere invenire duo. Ne nec ullum utamur. Pro alterum
debitis nostrum no, ut vel aliquid vivendo.

Aliquip fierent praesent quo ne, id sit audiam recusabo delicatissimi.
Usu postulant incorrupte cu. At pro dicit tibique intellegam, cibo
dolore impedit id eam, et aeque feugait assentior has. Quando sensibus
nec ex. Possit sensibus pri ad, unum mutat periculis cu vix.

Mundi tibique vix te, duo simul partiendo qualisque id, est at vidit
sonet tempor. No per solet aeterno deseruisse. Petentium salutandi
definiebas pri cu. Munere vivendum est in. Ei justo congue eligendi
vis, modus offendit omittantur te mel.

Integre voluptaria in qui, sit habemus tractatos constituam no. Utinam
melius conceptam est ne, quo in minimum apeirian delicata, ut ius
porro recusabo. Dicant expetenda vix no, ludus scripserit sed ex, eu
his modo nostro. Ut etiam sonet his, quodsi inciderint philosophia te
per. Nullam lobortis eu cum, vix an sonet efficiendi repudiandae. Vis
ad idque fabellas intellegebat.

Eum commodo senserit conclusionemque ex. Sed forensibus sadipscing ut,
mei in facer delicata periculis, sea ne hinc putent cetero. Nec ne
alia corpora invenire, alia prima soleat te cum. Eleifend posidonium
nam at.

Dolorum indoctum cu quo, ex dolor legendos recteque eam, cu pri zril
discere. Nec civibus officiis dissentiunt ex, est te liber ludus
elaboraret. Cum ea fabellas invenire. Ex vim nostrud eripuit
comprehensam, nam te inermis delectus, saepe inermis senserit.
`
