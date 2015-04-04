package gen

import (
	"io"
	"text/template"
)

var (
	marshalTestTempl = template.New("MarshalTest")
	encodeTestTempl  = template.New("EncodeTest")
)

// TODO(philhofer):
// for simplicity's sake, right now
// we can only generate tests for types
// that can be initialized with the
// "Type{}" syntax.
// we should support all the types.

type mtestGen struct{ io.Writer }

func (m mtestGen) Execute(p Elem) error {
	if IsPrintable(p) {
		switch p.(type) {
		case *Struct, *Array, *Slice, *Map:
			return marshalTestTempl.Execute(m.Writer, p)
		}
	}
	return nil
}

type etestGen struct{ io.Writer }

func (e etestGen) Execute(p Elem) error {
	if IsPrintable(p) {
		switch p.(type) {
		case *Struct, *Array, *Slice, *Map:
			return encodeTestTempl.Execute(e.Writer, p)
		}
	}
	return nil
}

func init() {
	template.Must(marshalTestTempl.Parse(`func Test{{.TypeName}}MarshalUnmarshal(t *testing.T) {
	v := {{.TypeName}}{}
	bts, err := v.MarshalMsg(nil)
	if err != nil {
		t.Fatal(err)
	}
	left, err := v.UnmarshalMsg(bts)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) > 0 {
		t.Errorf("%d bytes left over after UnmarshalMsg(): %q", len(left), left)
	}

	left, err = msgp.Skip(bts)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) > 0 {
		t.Errorf("%d bytes left over after Skip(): %q", len(left), left)
	}
}

func Benchmark{{.TypeName}}MarshalMsg(b *testing.B) {
	v := {{.TypeName}}{}
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		v.MarshalMsg(nil)
	}
}

func Benchmark{{.TypeName}}AppendMsg(b *testing.B) {
	v := {{.TypeName}}{}
	bts := make([]byte, 0, v.Msgsize())
	bts, _ = v.MarshalMsg(bts[0:0])
	b.SetBytes(int64(len(bts)))
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		bts, _ = v.MarshalMsg(bts[0:0])
	}
}

func Benchmark{{.TypeName}}Unmarshal(b *testing.B) {
	v := {{.TypeName}}{}
	bts, _ := v.MarshalMsg(nil)
	b.ReportAllocs()
	b.SetBytes(int64(len(bts)))
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		_, err := v.UnmarshalMsg(bts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

`))

	template.Must(encodeTestTempl.Parse(`func Test{{.TypeName}}EncodeDecode(t *testing.T) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer
	msgp.Encode(&buf, &v)

	m := v.Msgsize()
	if buf.Len() > m {
		t.Logf("WARNING: Msgsize() for %v is inaccurate", v)
	}

	vn := {{.TypeName}}{}
	err := msgp.Decode(&buf, &vn)
	if err != nil {
		t.Error(err)
	}

	buf.Reset()
	msgp.Encode(&buf, &v)
	err = msgp.NewReader(&buf).Skip()
	if err != nil {
		t.Error(err)
	}
}

func Benchmark{{.TypeName}}Encode(b *testing.B) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer 
	msgp.Encode(&buf, &v)
	b.SetBytes(int64(buf.Len()))
	en := msgp.NewWriter(msgp.Nowhere)
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		v.EncodeMsg(en)
	}
	en.Flush()
}

func Benchmark{{.TypeName}}Decode(b *testing.B) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer
	msgp.Encode(&buf, &v)
	b.SetBytes(int64(buf.Len()))
	rd := msgp.NewEndlessReader(buf.Bytes(), b)
	dc := msgp.NewReader(rd)
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		err := v.DecodeMsg(dc)
		if  err != nil {
			b.Fatal(err)
		}
	}
}

`))

}
