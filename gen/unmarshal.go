package gen

import (
	"strconv"
)

type unmarshalGen struct {
	p        printer
	hasfield bool
}

func (u *unmarshalGen) needsField() {
	if u.hasfield {
		return
	}
	u.p.print("\nvar field []byte; _ = field")
	u.hasfield = true
}

func (u *unmarshalGen) Execute(p Elem) error {
	u.hasfield = false
	if !u.p.ok() {
		return u.p.err
	}
	if !IsPrintable(p) {
		return nil
	}

	u.p.comment("UnmarshalMsg implements msgp.Unmarshaler")

	u.p.printf("\nfunc (%s %s) UnmarshalMsg(bts []byte) (o []byte, err error) {", p.Varname(), methodReceiver(p))
	next(u, p)
	u.p.print("\no = bts")
	u.p.nakedReturn()
	unsetReceiver(p)
	return u.p.err
}

// does assignment to the variable "name" with the type "base"
func (u *unmarshalGen) assignAndCheck(name string, base string) {
	if !u.p.ok() {
		return
	}
	u.p.printf("\n%s, bts, err = msgp.Read%sBytes(bts)", name, base)
	u.p.print(errcheck)
}

func (u *unmarshalGen) gStruct(s *Struct) {
	if !u.p.ok() {
		return
	}
	if s.AsTuple {
		u.tuple(s)
	} else {
		u.mapstruct(s)
	}
	return
}

func (u *unmarshalGen) tuple(s *Struct) {

	// open block
	u.p.print("\n{")
	u.p.declare(structArraySizeVar, u32)
	u.assignAndCheck(structArraySizeVar, arrayHeader)
	u.p.arrayCheck(strconv.Itoa(len(s.Fields)), structArraySizeVar)
	u.p.closeblock() // close 'ssz' block
	for i := range s.Fields {
		if !u.p.ok() {
			return
		}
		next(u, s.Fields[i].FieldElem)
	}
}

func (u *unmarshalGen) mapstruct(s *Struct) {
	u.needsField()
	u.p.declare(structMapSizeVar, u32)
	u.assignAndCheck(structMapSizeVar, mapHeader)

	u.p.print("\nfor isz > 0 {")
	u.p.print("\nisz--; field, bts, err = msgp.ReadMapKeyZC(bts)")
	u.p.print(errcheck)
	u.p.print("\nswitch msgp.UnsafeString(field) {")
	for i := range s.Fields {
		if !u.p.ok() {
			return
		}
		u.p.printf("\ncase \"%s\":", s.Fields[i].FieldTag)
		next(u, s.Fields[i].FieldElem)
	}
	u.p.print("\ndefault:\nbts, err = msgp.Skip(bts)")
	u.p.print(errcheck)
	u.p.print("\n}\n}") // close switch and for loop
}

func (u *unmarshalGen) gBase(b *BaseElem) {
	if !u.p.ok() {
		return
	}

	refname := b.Varname() // assigned to
	lowered := b.Varname() // passed as argument
	if b.Convert {
		// begin 'tmp' block
		refname = "tmp"
		lowered = b.ToBase() + "(" + lowered + ")"
		u.p.printf("\n{\nvar tmp %s", b.BaseType())
	}

	switch b.Value {
	case Bytes:
		u.p.printf("\n%s, bts, err = msgp.ReadBytesBytes(bts, %s)", refname, lowered)
	case Ext:
		u.p.printf("\nbts, err = msgp.ReadExtensionBytes(bts, %s)", lowered)
	case IDENT:
		u.p.printf("\nbts, err = %s.UnmarshalMsg(bts)", lowered)
	default:
		u.p.printf("\n%s, bts, err = msgp.Read%sBytes(bts)", refname, b.BaseName())
	}
	if b.Convert {
		// close 'tmp' block
		u.p.printf("\n%s = %s(tmp)\n}", b.Varname(), b.FromBase())
	}

	u.p.print(errcheck)
}

func (u *unmarshalGen) gArray(a *Array) {
	if !u.p.ok() {
		return
	}

	// special case for [const]byte objects
	// see decode.go for symmetry
	if be, ok := a.Els.(*BaseElem); ok && be.Value == Byte {
		u.p.printf("\nbts, err = msgp.ReadExactBytes(bts, %s[:])", a.Varname())
		u.p.print(errcheck)
		return
	}

	u.p.declare(arraySizeVar, u32)
	u.assignAndCheck(arraySizeVar, arrayHeader)
	u.p.arrayCheck(a.Size, arraySizeVar)
	u.p.rangeBlock(a.Index, a.Varname(), u, a.Els)
}

func (u *unmarshalGen) gSlice(s *Slice) {
	if !u.p.ok() {
		return
	}
	u.p.declare(sliceSizeVar, u32)
	u.assignAndCheck(sliceSizeVar, arrayHeader)
	u.p.resizeSlice(sliceSizeVar, s)
	u.p.rangeBlock(s.Index, s.Varname(), u, s.Els)
}

func (u *unmarshalGen) gMap(m *Map) {
	if !u.p.ok() {
		return
	}
	u.p.declare(mapSizeVar, u32)
	u.assignAndCheck(mapSizeVar, mapHeader)

	// allocate or clear map
	u.p.resizeMap(mapSizeVar, m)

	// loop and get key,value
	u.p.print("\nfor msz > 0 {")
	u.p.printf("\nvar %s string; var %s %s; msz--", m.Keyidx, m.Validx, m.Value.TypeName())
	u.assignAndCheck(m.Keyidx, stringTyp)
	next(u, m.Value)
	u.p.mapAssign(m)
	u.p.closeblock()
}

func (u *unmarshalGen) gPtr(p *Ptr) {
	u.p.printf("\nif msgp.IsNil(bts) { bts, err = msgp.ReadNilBytes(bts); if err != nil { return }; %s = nil; } else { ", p.Varname())
	u.p.initPtr(p)
	next(u, p.Value)
	u.p.closeblock()
}
