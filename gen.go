package typegen

import (
	"container/list"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"text/template"

	"github.com/ipfs/go-cid"
)

const MaxLength = 8192

const ByteArrayMaxLen = 2 << 20

var (
	cidType      = reflect.TypeOf(cid.Cid{})
	bigIntType   = reflect.TypeOf(big.Int{})
	deferredType = reflect.TypeOf(Deferred{})
)

func doTemplate(w io.Writer, info interface{}, templ string) error {
	t := template.Must(template.New("").
		Funcs(template.FuncMap{
			"MajorType": func(wname string, tname string, val string) string {
				return fmt.Sprintf(`if err := cbg.WriteMajorTypeHeaderBuf(scratch, %s, %s, uint64(%s)); err != nil {
	return err
}`, wname, tname, val)
			},
			"ReadHeader": func(rdr string) string {
				return fmt.Sprintf(`cbg.CborReadHeaderBuf(%s, scratch)`, rdr)
			},
		}).Parse(templ))

	return t.Execute(w, info)
}

func PrintHeaderAndUtilityMethods(w io.Writer, pkg string, typeInfos []*GenTypeInfo) error {
	var imports []Import
	for _, gti := range typeInfos {
		imports = append(imports, gti.Imports()...)
	}

	imports = append(imports, defaultImports...)
	imports = dedupImports(imports)

	data := struct {
		Package string
		Imports []Import
	}{pkg, imports}
	return doTemplate(w, data, `// Code generated by github.com/daotl/cbor-gen. DO NOT EDIT.

package {{ .Package }}

import (
	"fmt"
	"io"
	"math"
	"sort"

{{ range .Imports }}{{ .Name }} "{{ .PkgPath }}"
{{ end }}
)


var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

`)
}

func emitInitNilEmbeddedStructMethod(w io.Writer, gti *GenTypeInfo,
	embeddedByPointerStructs []string) error {
	data := struct {
		Name   string
		Embeds []string
	}{gti.Name, embeddedByPointerStructs}
	return doTemplate(w, data, `
func (t *{{ .Name }}) InitNilEmbeddedStruct() {
	if t != nil {
		{{ range .Embeds }}if t.{{ . }} == nil {
			t.{{ . }} = &{{ . }}{}
		}
		t.{{ . }}.InitNilEmbeddedStruct(){{ end }}	}
}

`)
}

type Field struct {
	Name    string
	MapKey  string
	Pointer bool
	Type    reflect.Type
	Pkg     string

	IterLabel string
}

func typeName(pkg string, t reflect.Type) string {
	switch t.Kind() {
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), typeName(pkg, t.Elem()))
	case reflect.Slice:
		return "[]" + typeName(pkg, t.Elem())
	case reflect.Ptr:
		return "*" + typeName(pkg, t.Elem())
	case reflect.Map:
		return "map[" + typeName(pkg, t.Key()) + "]" + typeName(pkg, t.Elem())
	default:
		pkgPath := t.PkgPath()
		if pkgPath == "" {
			// It's a built-in.
			return t.String()
		} else if pkgPath == pkg {
			return t.Name()
		}
		return fmt.Sprintf("%s.%s", resolvePkgName(pkgPath, t.String()), t.Name())
	}
}

func (f Field) TypeName() string {
	return typeName(f.Pkg, f.Type)
}

func (f Field) ElemName() string {
	return typeName(f.Pkg, f.Type.Elem())
}

func (f Field) IsArray() bool {
	return f.Type.Kind() == reflect.Array
}

func (f Field) Len() int {
	return f.Type.Len()
}

type GenTypeInfo struct {
	Name   string
	Fields []Field
}

func (gti *GenTypeInfo) Imports() []Import {
	var imports []Import
	for _, f := range gti.Fields {
		switch f.Type.Kind() {
		case reflect.Struct:
			if !f.Pointer && f.Type != bigIntType {
				continue
			}
			if f.Type == cidType {
				continue
			}
		case reflect.Bool:
			continue
		}
		imports = append(imports, ImportsForType(f.Pkg, f.Type)...)
	}
	return imports
}

func (gti *GenTypeInfo) NeedsScratch() bool {
	for _, f := range gti.Fields {
		switch f.Type.Kind() {
		case reflect.String,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Array,
			reflect.Slice,
			reflect.Map:
			return true

		case reflect.Struct:
			if f.Type == bigIntType || f.Type == cidType {
				return true
			}
			// nope
		case reflect.Bool:
			// nope
		}
	}
	return false
}

func nameIsExported(name string) bool {
	return strings.ToUpper(name[0:1]) == name[0:1]
}

func ParseTypeInfo(i interface{}, flattenEmbeddedStruct bool) (
	gti *GenTypeInfo, embeddedByPointerStructs *[]string, err error) {
	t := reflect.TypeOf(i)

	pkg := t.PkgPath()

	fields := list.New()
	fieldMap := map[string]*list.Element{}
	embeddedByPointerStructs = &[]string{}
	err = parseTypeInfoRecur(pkg, t, flattenEmbeddedStruct, 0, fields, fieldMap,
		map[string]int{}, embeddedByPointerStructs)

	gti = &GenTypeInfo{
		Name: t.Name(),
	}
	for e := fields.Front(); e != nil; e = e.Next() {
		gti.Fields = append(gti.Fields, e.Value.(Field))
	}

	return gti, embeddedByPointerStructs, err
}

func parseTypeInfoRecur(pkg string, t reflect.Type, flattenEmbeddedStruct bool,
	depth int, fields *list.List, fieldMap map[string]*list.Element,
	depths map[string]int, embeddedByPointerStructs *[]string) error {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !nameIsExported(f.Name) {
			continue
		}

		ft := f.Type
		var pointer bool
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
			pointer = true
		}

		if flattenEmbeddedStruct && ft.Kind() == reflect.Struct && f.Anonymous {
			// Only need to record first-level structs embedded by pointer,
			// for later generating pointer initialization code
			if depth == 0 && pointer {
				*embeddedByPointerStructs = append(*embeddedByPointerStructs, f.Name)
			}
			parseTypeInfoRecur(pkg, ft, true, depth+1, fields, fieldMap, depths, nil)
		} else {
			if flattenEmbeddedStruct {
				// If a previous field of the same name has a greater depth, skip
				if prevDepth, ok := depths[f.Name]; ok && prevDepth < depth {
					continue
				}
				depths[f.Name] = depth
				// Remove the previous field of the same name from the list,
				// since the new one overrides it
				if e, ok := fieldMap[f.Name]; ok {
					fields.Remove(e)
				}
			}

			mapk := f.Name
			tagval := f.Tag.Get("cborgen")
			if tagval != "" {
				mapk = tagval
			}

			f := Field{
				Name:    f.Name,
				MapKey:  mapk,
				Pointer: pointer,
				Type:    ft,
				Pkg:     pkg,
			}
			// Push the new field to the back of the list
			fieldMap[f.Name] = fields.PushBack(f)
		}
	}

	return nil
}

func (gti GenTypeInfo) TupleHeader() []byte {
	return CborEncodeMajorType(MajArray, uint64(len(gti.Fields)))
}

func (gti GenTypeInfo) TupleHeaderAsByteString() string {
	h := gti.TupleHeader()
	s := "[]byte{"
	for _, b := range h {
		s += fmt.Sprintf("%d,", b)
	}
	s += "}"
	return s
}

func (gti GenTypeInfo) MapHeader() []byte {
	return CborEncodeMajorType(MajMap, uint64(len(gti.Fields)))
}

func (gti GenTypeInfo) MapHeaderAsByteString() string {
	h := gti.MapHeader()
	s := "[]byte{"
	for _, b := range h {
		s += fmt.Sprintf("%d,", b)
	}
	s += "}"
	return s
}

func emitCborMarshalStringField(w io.Writer, f Field) error {
	if f.Pointer {
		return fmt.Errorf("pointers to strings not supported")
	}

	return doTemplate(w, f, `
	if len({{ .Name }}) > cbg.MaxLength {
		return xerrors.Errorf("Value in field {{ .Name | js }} was too long")
	}

	{{ MajorType "w" "cbg.MajTextString" (print "len(" .Name ")") }}
	if _, err := io.WriteString(w, string({{ .Name }})); err != nil {
		return err
	}
`)
}
func emitCborMarshalStructField(w io.Writer, f Field) error {
	switch f.Type {
	case bigIntType:
		return doTemplate(w, f, `
	{
		if err := cbg.CborWriteHeader(w, cbg.MajTag, 2); err != nil {
			return err
		}
		var b []byte
		if {{ .Name }} != nil {
			b = {{ .Name }}.Bytes()
		}

		if err := cbg.CborWriteHeader(w, cbg.MajByteString, uint64(len(b))); err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
`)

	case cidType:
		return doTemplate(w, f, `
{{ if .Pointer }}
	if {{ .Name }} == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCidBuf(scratch, w, *{{ .Name }}); err != nil {
			return xerrors.Errorf("failed to write cid field {{ .Name }}: %w", err)
		}
	}
{{ else }}
	if err := cbg.WriteCidBuf(scratch, w, {{ .Name }}); err != nil {
		return xerrors.Errorf("failed to write cid field {{ .Name }}: %w", err)
	}
{{ end }}
`)
	default:
		return doTemplate(w, f, `
	if err := {{ .Name }}.MarshalCBOR(w); err != nil {
		return err
	}
`)
	}

}

func emitCborMarshalUint64Field(w io.Writer, f Field) error {
	return doTemplate(w, f, `
{{ if .Pointer }}
	if {{ .Name }} == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		{{ MajorType "w" "cbg.MajUnsignedInt" (print "*" .Name) }}
	}
{{ else }}
	{{ MajorType "w" "cbg.MajUnsignedInt" .Name }}
{{ end }}
`)
}

func emitCborMarshalUintField(w io.Writer, f Field) error {
	if f.Pointer {
		return fmt.Errorf("pointers to integers not supported")
	}
	return doTemplate(w, f, `
{{ MajorType "w" "cbg.MajUnsignedInt" .Name }}
`)
}

func emitCborMarshalIntField(w io.Writer, f Field) error {
	if f.Pointer {
		return fmt.Errorf("pointers to integers not supported")
	}

	// if negative
	// val = -1 - cbor
	// cbor = -val -1

	return doTemplate(w, f, `
	if {{ .Name }} >= 0 {
	{{ MajorType "w" "cbg.MajUnsignedInt" .Name }}
	} else {
	{{ MajorType "w" "cbg.MajNegativeInt" (print "-" .Name "-1") }}
	}
`)
}

func emitCborMarshalBoolField(w io.Writer, f Field) error {
	return doTemplate(w, f, `
	if err := cbg.WriteBool(w, {{ .Name }}); err != nil {
		return err
	}
`)
}

func emitCborMarshalMapField(w io.Writer, f Field) error {
	err := doTemplate(w, f, `
{
	if len({{ .Name }}) > 4096 {
		return xerrors.Errorf("cannot marshal {{ .Name }} map too large")
	}

	{{ MajorType "w" "cbg.MajMap" (print "len(" .Name ")") }}

	keys := make([]string, 0, len({{ .Name }}))
	for k := range {{ .Name }} {
		keys = append(keys, k)
	}
	cbg.MapKeySort_RFC7049(keys)
	for _, k := range keys {
		v := {{ .Name }}[k]

`)
	if err != nil {
		return err
	}

	// Map key
	switch f.Type.Key().Kind() {
	case reflect.String:
		if err := emitCborMarshalStringField(w, Field{Name: "k"}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("non-string map keys are not yet supported")
	}

	// Map value
	switch f.Type.Elem().Kind() {
	case reflect.Ptr:
		if f.Type.Elem().Elem().Kind() != reflect.Struct {
			return fmt.Errorf("unsupported map elem ptr type: %s", f.Type.Elem())
		}

		fallthrough
	case reflect.Struct:
		if err := emitCborMarshalStructField(w, Field{Name: "v", Type: f.Type.Elem(), Pkg: f.Pkg}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("currently unsupported map elem type: %s", f.Type.Elem())
	}

	return doTemplate(w, f, `
	}
	}
`)
}

func emitCborMarshalSliceField(w io.Writer, f Field) error {
	if f.Pointer {
		return fmt.Errorf("pointers to slices not supported")
	}
	e := f.Type.Elem()

	// Note: this re-slices the slice to deal with arrays.
	if e.Kind() == reflect.Uint8 || e.Kind() == reflect.Int8 {
		return doTemplate(w, f, `
	if len({{ .Name }}) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field {{ .Name }} was too long")
	}

	{{ MajorType "w" "cbg.MajByteString" (print "len(" .Name ")" ) }}

	if _, err := w.Write({{ .Name }}[:]); err != nil {
		return err
	}
`)
	}

	if e.Kind() == reflect.Ptr {
		e = e.Elem()
	}

	err := doTemplate(w, f, `
	if len({{ .Name }}) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field {{ .Name }} was too long")
	}

	{{ MajorType "w" "cbg.MajArray" ( print "len(" .Name ")" ) }}
	for _, v := range {{ .Name }} {`)
	if err != nil {
		return err
	}

	switch e.Kind() {
	default:
		return fmt.Errorf("do not yet support slices of %s yet", e.Kind())
	case reflect.Struct:
		switch e {
		case cidType:
			err := doTemplate(w, f, `
		if err := cbg.WriteCidBuf(scratch, w, v); err != nil {
			return xerrors.Errorf("failed writing cid field {{ .Name }}: %w", err)
		}
`)
			if err != nil {
				return err
			}

		default:
			err := doTemplate(w, f, `
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
`)
			if err != nil {
				return err
			}
		}
	case reflect.Uint64:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint8:
		err := doTemplate(w, f, `
		if err := cbg.CborWriteHeader(w, cbg.MajUnsignedInt, uint64(v)); err != nil {
			return err
		}
`)
		if err != nil {
			return err
		}
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
		subf := Field{Name: "v", Type: e, Pkg: f.Pkg}
		if err := emitCborMarshalIntField(w, subf); err != nil {
			return err
		}

	case reflect.Slice:
		subf := Field{Name: "v", Type: e, Pkg: f.Pkg}
		if err := emitCborMarshalSliceField(w, subf); err != nil {
			return err
		}
	}

	// array end
	fmt.Fprintf(w, "\t}\n")
	return nil
}

func emitCborMarshalStructTuple(w io.Writer, gti *GenTypeInfo, flattenEmbeddedStruct bool) error {
	// 9 byte buffer to accomodate for the maximum header length (cbor varints are maximum 9 bytes_
	err := doTemplate(w, gti, `var lengthBuf{{ .Name }} = {{ .TupleHeaderAsByteString }}
func (t *{{ .Name }}) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}`)
	if err != nil {
		return err
	}

	if flattenEmbeddedStruct {
		err = doTemplate(w, gti, `
	t.InitNilEmbeddedStruct()`)
	}
	if err != nil {
		return err
	}

	err = doTemplate(w, gti, `
	if _, err := w.Write(lengthBuf{{ .Name }}); err != nil {
		return err
	}
{{ if .NeedsScratch }}
	scratch := make([]byte, 9)
{{ end }}
`)
	if err != nil {
		return err
	}

	for _, f := range gti.Fields {
		fmt.Fprintf(w, "\n\t// t.%s (%s) (%s)", f.Name, f.Type, f.Type.Kind())
		f.Name = "t." + f.Name

		switch f.Type.Kind() {
		case reflect.String:
			if err := emitCborMarshalStringField(w, f); err != nil {
				return err
			}
		case reflect.Struct:
			if err := emitCborMarshalStructField(w, f); err != nil {
				return err
			}
		case reflect.Uint64:
			if err := emitCborMarshalUint64Field(w, f); err != nil {
				return err
			}
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			if err := emitCborMarshalUintField(w, f); err != nil {
				return err
			}
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			if err := emitCborMarshalIntField(w, f); err != nil {
				return err
			}
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			if err := emitCborMarshalSliceField(w, f); err != nil {
				return err
			}
		case reflect.Bool:
			if err := emitCborMarshalBoolField(w, f); err != nil {
				return err
			}
		case reflect.Map:
			if err := emitCborMarshalMapField(w, f); err != nil {
				return err
			}
		default:
			return fmt.Errorf("field %q of %q has unsupported kind %q", f.Name, gti.Name, f.Type.Kind())
		}
	}

	fmt.Fprintf(w, "\treturn nil\n}\n\n")
	return nil
}

func emitCborUnmarshalStringField(w io.Writer, f Field) error {
	if f.Pointer {
		return fmt.Errorf("pointers to strings not supported")
	}
	if f.Type == nil {
		f.Type = reflect.TypeOf("")
	}
	return doTemplate(w, f, `
	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		{{ .Name }} = {{ .TypeName }}(sval)
	}
`)
}

func emitCborUnmarshalStructField(w io.Writer, f Field) error {
	switch f.Type {
	case bigIntType:
		return doTemplate(w, f, `
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}

	if maj != cbg.MajTag || extra != 2 {
		return fmt.Errorf("big ints should be cbor bignums")
	}

	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}

	if maj != cbg.MajByteString {
		return fmt.Errorf("big ints should be tagged cbor byte strings")
	}

	if extra > 256 {
		return fmt.Errorf("{{ .Name }}: cbor bignum was too large")
	}

	if extra > 0 {
		buf := make([]byte, extra)
		if _, err := io.ReadFull(br, buf); err != nil {
			return err
		}
		{{ .Name }} = big.NewInt(0).SetBytes(buf)
	} else {
		{{ .Name }} = big.NewInt(0)
	}
`)
	case cidType:
		return doTemplate(w, f, `
	{
{{ if .Pointer }}
		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		if b != cbg.CborNull[0] {
			if err := br.UnreadByte(); err != nil {
				return err
			}
{{ end }}
		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field {{ .Name }}: %w", err)
		}
{{ if .Pointer }}
			{{ .Name }} = &c
		}
{{ else }}
		{{ .Name }} = c
{{ end }}
	}
`)
	case deferredType:
		return doTemplate(w, f, `
	{
{{ if .Pointer }}
		{{ .Name }} = new(cbg.Deferred)
{{ end }}
		if err := {{ .Name }}.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("failed to read deferred field: %w", err)
		}
	}
`)

	default:
		return doTemplate(w, f, `
	{
{{ if .Pointer }}
		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		if b != cbg.CborNull[0] {
			if err := br.UnreadByte(); err != nil {
				return err
			}
			{{ .Name }} = new({{ .TypeName }})
			if err := {{ .Name }}.UnmarshalCBOR(br); err != nil {
				return xerrors.Errorf("unmarshaling {{ .Name }} pointer: %w", err)
			}
		}
{{ else }}
		if err := {{ .Name }}.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling {{ .Name }}: %w", err)
		}
{{ end }}
	}
`)
	}
}

func emitCborUnmarshalIntField(w io.Writer, f Field, len int) error {
	return doTemplate(w, f, fmt.Sprintf(`{
	maj, extra, err := {{ ReadHeader "br" }}
	var extraI int%d
	if err != nil {
		return err
	}
	switch maj {
	case cbg.MajUnsignedInt:
		extraI = int%d(extra)
		if extraI < 0 {
			return fmt.Errorf("int%d positive overflow")
	   }
	case cbg.MajNegativeInt:
		extraI = int%d(extra)
		if extraI < 0 {
			return fmt.Errorf("int%d negative oveflow")
		}
		extraI = -1 - extraI
	default:
		return fmt.Errorf("wrong type for int%d field: %%d", maj)
	}

	{{ .Name }} = {{ .TypeName }}(extraI)
}
`, len, len, len, len, len, len))
}

func emitCborUnmarshalUint64Field(w io.Writer, f Field) error {
	return doTemplate(w, f, `
	{
{{ if .Pointer }}
	b, err := br.ReadByte()
	if err != nil {
		return err
	}
	if b != cbg.CborNull[0] {
		if err := br.UnreadByte(); err != nil {
			return err
		}
		maj, extra, err = {{ ReadHeader "br" }}
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		typed := {{ .TypeName }}(extra)
		{{ .Name }} = &typed
	}
{{ else }}
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	{{ .Name }} = {{ .TypeName }}(extra)
{{ end }}
	}
`)
}

func emitCborUnmarshalUintField(w io.Writer, f Field, len int) error {
	return doTemplate(w, f, fmt.Sprintf(`
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint%d field")
	}
	if extra > math.MaxUint%d {
		return fmt.Errorf("integer in input was too large for uint%d field")
	}
	{{ .Name }} = {{ .TypeName }}(extra)
`, len, len, len))
}

func emitCborUnmarshalBoolField(w io.Writer, f Field) error {
	return doTemplate(w, f, `
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajOther {
		return fmt.Errorf("booleans must be major type 7")
	}
	switch extra {
	case 20:
		{{ .Name }} = false
	case 21:
		{{ .Name }} = true
	default:
		return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
	}
`)
}

func emitCborUnmarshalMapField(w io.Writer, f Field) error {
	err := doTemplate(w, f, `
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("expected a map (major type 5)")
	}
	if extra > 4096 {
		return fmt.Errorf("{{ .Name }}: map too large")
	}

	{{ .Name }} = make({{ .TypeName }}, extra)


	for i, l := 0, int(extra); i < l; i++ {
`)
	if err != nil {
		return err
	}

	switch f.Type.Key().Kind() {
	case reflect.String:
		if err := doTemplate(w, f, `
	var k string
`); err != nil {
			return err
		}
		if err := emitCborUnmarshalStringField(w, Field{Name: "k"}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("maps with non-string keys are not yet supported")
	}

	var pointer bool
	t := f.Type.Elem()
	switch t.Kind() {
	case reflect.Ptr:
		if t.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("unsupported map elem ptr type: %s", t)
		}

		pointer = true
		fallthrough
	case reflect.Struct:
		subf := Field{Name: "v", Pointer: pointer, Type: t, Pkg: f.Pkg}
		if err := doTemplate(w, subf, `
	var v {{ .TypeName }}
`); err != nil {
			return err
		}

		if pointer {
			subf.Type = subf.Type.Elem()
		}
		if err := emitCborUnmarshalStructField(w, subf); err != nil {
			return err
		}
		if err := doTemplate(w, f, `
	{{ .Name }}[k] = v
`); err != nil {
			return err
		}
	default:
		return fmt.Errorf("currently only support maps of structs")
	}

	return doTemplate(w, f, `
	}
`)
}

func emitCborUnmarshalSliceField(w io.Writer, f Field) error {
	if f.IterLabel == "" {
		f.IterLabel = "i"
	}

	e := f.Type.Elem()
	var pointer bool
	if e.Kind() == reflect.Ptr {
		pointer = true
		e = e.Elem()
	}

	err := doTemplate(w, f, `
	maj, extra, err = {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
`)
	if err != nil {
		return err
	}

	if e.Kind() == reflect.Uint8 || e.Kind() == reflect.Int8 {
		return doTemplate(w, f, `
	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("{{ .Name }}: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	{{if .IsArray}}
	if extra != {{ .Len }} {
		return fmt.Errorf("expected array to have {{ .Len }} elements")
	}

	{{ .Name }} = {{ .TypeName }}{}
	{{else}}
	if extra > 0 {
		{{ .Name }} = make({{ .TypeName }}, extra)
	}
	{{end}}
	if _, err := io.ReadFull(br, {{ .Name }}[:]); err != nil {
		return err
	}
`)
	}

	if err := doTemplate(w, f, `
	if extra > cbg.MaxLength {
		return fmt.Errorf("{{ .Name }}: array too large (%d)", extra)
	}
`); err != nil {
		return err
	}

	err = doTemplate(w, f, `
	if maj != cbg.MajArray {
		return fmt.Errorf("expected cbor array")
	}
	{{if .IsArray}}
	if extra != {{ .Len }} {
		return fmt.Errorf("expected array to have {{ .Len }} elements")
	}

	{{ .Name }} = {{ .TypeName }}{}
	{{else}}
	if extra > 0 {
		{{ .Name }} = make({{ .TypeName }}, extra)
	}
	{{end}}
	for {{ .IterLabel }} := 0; {{ .IterLabel }} < int(extra); {{ .IterLabel }}++ {
`)
	if err != nil {
		return err
	}

	len := 0
	switch e.Kind() {
	case reflect.Struct:
		fname := e.PkgPath() + "." + e.Name()
		switch fname {
		case "github.com/ipfs/go-cid.Cid":
			err := doTemplate(w, f, `
		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("reading cid field {{ .Name }} failed: %w", err)
		}
		{{ .Name }}[{{ .IterLabel }}] = c
`)
			if err != nil {
				return err
			}
		default:
			subf := Field{
				Type:    e,
				Pkg:     f.Pkg,
				Pointer: pointer,
				Name:    f.Name + "[" + f.IterLabel + "]",
			}

			err := doTemplate(w, subf, `
		var v {{ .TypeName }}
		if err := v.UnmarshalCBOR(br); err != nil {
			return err
		}

		{{ .Name }} = {{ if .Pointer }}&{{ end }}v
`)
			if err != nil {
				return err
			}
		}
	case reflect.Uint16:
		if len == 0 {
			len = 16
		}
		fallthrough
	case reflect.Uint32:
		if len == 0 {
			len = 32
		}
		fallthrough
	case reflect.Uint64:
		if len == 0 {
			len = 64
		}
		err := doTemplate(w, f, fmt.Sprintf(`
		maj, val, err := {{ ReadHeader "br" }}
		if err != nil {
			return xerrors.Errorf("failed to read uint%d for {{ .Name }} slice: %%w", err)
		}

		if maj != cbg.MajUnsignedInt {
			return xerrors.Errorf("value read for array {{ .Name }} was not a uint, instead got %%d", maj)
		}
		
		{{ .Name }}[{{ .IterLabel}}] = {{ .ElemName }}(val)
`, len))
		if err != nil {
			return err
		}
	case reflect.Int8:
		if len == 0 {
			len = 8
		}
		fallthrough
	case reflect.Int16:
		if len == 0 {
			len = 16
		}
		fallthrough
	case reflect.Int32:
		if len == 0 {
			len = 32
		}
		fallthrough
	case reflect.Int64:
		if len == 0 {
			len = 64
		}
		subf := Field{
			Type: e,
			Pkg:  f.Pkg,
			Name: f.Name + "[" + f.IterLabel + "]",
		}
		err := emitCborUnmarshalIntField(w, subf, len)
		if err != nil {
			return err
		}
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		nextIter := string([]byte{f.IterLabel[0] + 1})
		subf := Field{
			Name:      fmt.Sprintf("%s[%s]", f.Name, f.IterLabel),
			Type:      e,
			IterLabel: nextIter,
			Pkg:       f.Pkg,
		}
		fmt.Fprintf(w, "\t\t{\n\t\t\tvar maj byte\n\t\tvar extra uint64\n\t\tvar err error\n")
		if err := emitCborUnmarshalSliceField(w, subf); err != nil {
			return err
		}
		fmt.Fprintf(w, "\t\t}\n")

	default:
		return fmt.Errorf("do not yet support slices of %s yet", e.Elem())
	}
	fmt.Fprintf(w, "\t}\n\n")

	return nil
}

func emitCborUnmarshalStructTuple(w io.Writer, gti *GenTypeInfo,
	flattenEmbeddedStruct bool) error {
	err := doTemplate(w, gti, `
func (t *{{ .Name}}) UnmarshalCBOR(r io.Reader) error {
	*t = {{.Name}}{}`)
	if err != nil {
		return err
	}

	if flattenEmbeddedStruct {
		err = doTemplate(w, gti, `
	t.InitNilEmbeddedStruct()`)
	}
	if err != nil {
		return err
	}

	err = doTemplate(w, gti, `

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != {{ len .Fields }} {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

`)
	if err != nil {
		return err
	}

	len := 0
	for _, f := range gti.Fields {
		fmt.Fprintf(w, "\t// t.%s (%s) (%s)\n", f.Name, f.Type, f.Type.Kind())
		f.Name = "t." + f.Name

		switch f.Type.Kind() {
		case reflect.String:
			if err := emitCborUnmarshalStringField(w, f); err != nil {
				return err
			}
		case reflect.Struct:
			if err := emitCborUnmarshalStructField(w, f); err != nil {
				return err
			}
		case reflect.Uint64:
			if err := emitCborUnmarshalUint64Field(w, f); err != nil {
				return err
			}
		case reflect.Uint8:
			if len == 0 {
				len = 8
			}
			fallthrough
		case reflect.Uint16:
			if len == 0 {
				len = 16
			}
			fallthrough
		case reflect.Uint32:
			if len == 0 {
				len = 32
			}
			if err := emitCborUnmarshalUintField(w, f, len); err != nil {
				return err
			}
		case reflect.Int8:
			if len == 0 {
				len = 8
			}
			fallthrough
		case reflect.Int16:
			if len == 0 {
				len = 16
			}
			fallthrough
		case reflect.Int32:
			if len == 0 {
				len = 32
			}
			fallthrough
		case reflect.Int64:
			if len == 0 {
				len = 64
			}
			if err := emitCborUnmarshalIntField(w, f, len); err != nil {
				return err
			}
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			if err := emitCborUnmarshalSliceField(w, f); err != nil {
				return err
			}
		case reflect.Bool:
			if err := emitCborUnmarshalBoolField(w, f); err != nil {
				return err
			}
		case reflect.Map:
			if err := emitCborUnmarshalMapField(w, f); err != nil {
				return err
			}
		default:
			return fmt.Errorf("field %q of %q has unsupported kind %q", f.Name, gti.Name, f.Type.Kind())
		}
	}

	fmt.Fprintf(w, "\treturn nil\n}\n\n")

	return nil
}

// Generates 'tuple representation' cbor encoders for the given type
func GenTupleEncodersForType(gti *GenTypeInfo, flattenEmbeddedStruct bool,
	embeddedByPointerStructs *[]string, w io.Writer) error {
	if flattenEmbeddedStruct {
		if err := emitInitNilEmbeddedStructMethod(w, gti, *embeddedByPointerStructs); err != nil {
			return err
		}
	}

	if err := emitCborMarshalStructTuple(w, gti, flattenEmbeddedStruct); err != nil {
		return err
	}

	if err := emitCborUnmarshalStructTuple(w, gti, flattenEmbeddedStruct); err != nil {
		return err
	}

	return nil
}

func emitCborMarshalStructMap(w io.Writer, gti *GenTypeInfo, flattenEmbeddedStruct bool) error {
	err := doTemplate(w, gti, `func (t *{{ .Name }}) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}`)
	if err != nil {
		return err
	}

	if flattenEmbeddedStruct {
		err = doTemplate(w, gti, `
	t.InitNilEmbeddedStruct()`)
	}
	if err != nil {
		return err
	}

	err = doTemplate(w, gti, `
	if _, err := w.Write({{ .MapHeaderAsByteString }}); err != nil {
		return err
	}

	scratch := make([]byte, 9)
`)
	if err != nil {
		return err
	}

	for _, f := range gti.Fields {
		fmt.Fprintf(w, "\n\t// t.%s (%s) (%s)", f.Name, f.Type, f.Type.Kind())

		if err := emitCborMarshalStringField(w, Field{
			Name: `"` + f.MapKey + `"`,
		}); err != nil {
			return err
		}

		f.Name = "t." + f.Name

		switch f.Type.Kind() {
		case reflect.String:
			if err := emitCborMarshalStringField(w, f); err != nil {
				return err
			}
		case reflect.Struct:
			if err := emitCborMarshalStructField(w, f); err != nil {
				return err
			}
		case reflect.Uint64:
			if err := emitCborMarshalUint64Field(w, f); err != nil {
				return err
			}
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			if err := emitCborMarshalIntField(w, f); err != nil {
				return err
			}
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			if err := emitCborMarshalUintField(w, f); err != nil {
				return err
			}
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			if err := emitCborMarshalSliceField(w, f); err != nil {
				return err
			}
		case reflect.Bool:
			if err := emitCborMarshalBoolField(w, f); err != nil {
				return err
			}
		case reflect.Map:
			if err := emitCborMarshalMapField(w, f); err != nil {
				return err
			}
		default:
			return fmt.Errorf("field %q of %q has unsupported kind %q", f.Name, gti.Name, f.Type.Kind())
		}
	}

	fmt.Fprintf(w, "\treturn nil\n}\n\n")
	return nil
}

func emitCborUnmarshalStructMap(w io.Writer, gti *GenTypeInfo,
	flattenEmbeddedStruct bool) error {
	err := doTemplate(w, gti, `
func (t *{{ .Name}}) UnmarshalCBOR(r io.Reader) error {
	*t = {{.Name}}{}`)
	if err != nil {
		return err
	}

	if flattenEmbeddedStruct {
		err = doTemplate(w, gti, `
	t.InitNilEmbeddedStruct()`)
	}
	if err != nil {
		return err
	}

	err = doTemplate(w, gti, `

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := {{ ReadHeader "br" }}
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("{{ .Name }}: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {
`)
	if err != nil {
		return err
	}

	if err := emitCborUnmarshalStringField(w, Field{Name: "name"}); err != nil {
		return err
	}

	err = doTemplate(w, gti, `
		switch name {
`)
	if err != nil {
		return err
	}

	for _, f := range gti.Fields {
		fmt.Fprintf(w, "// t.%s (%s) (%s)", f.Name, f.Type, f.Type.Kind())

		err := doTemplate(w, f, `
		case "{{ .MapKey }}":
`)
		if err != nil {
			return err
		}

		f.Name = "t." + f.Name

		len := 0
		switch f.Type.Kind() {
		case reflect.String:
			if err := emitCborUnmarshalStringField(w, f); err != nil {
				return err
			}
		case reflect.Struct:
			if err := emitCborUnmarshalStructField(w, f); err != nil {
				return err
			}
		case reflect.Uint64:
			if err := emitCborUnmarshalUint64Field(w, f); err != nil {
				return err
			}
		case reflect.Int8:
			if len == 0 {
				len = 8
			}
			fallthrough
		case reflect.Int16:
			if len == 0 {
				len = 16
			}
			fallthrough
		case reflect.Int32:
			if len == 0 {
				len = 32
			}
			fallthrough
		case reflect.Int64:
			if len == 0 {
				len = 64
			}
			if err := emitCborUnmarshalIntField(w, f, len); err != nil {
				return err
			}
		case reflect.Uint8:
			if len == 0 {
				len = 8
			}
			fallthrough
		case reflect.Uint16:
			if len == 0 {
				len = 16
			}
			fallthrough
		case reflect.Uint32:
			if len == 0 {
				len = 32
			}
			if err := emitCborUnmarshalUintField(w, f, len); err != nil {
				return err
			}
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			if err := emitCborUnmarshalSliceField(w, f); err != nil {
				return err
			}
		case reflect.Bool:
			if err := emitCborUnmarshalBoolField(w, f); err != nil {
				return err
			}
		case reflect.Map:
			if err := emitCborUnmarshalMapField(w, f); err != nil {
				return err
			}
		default:
			return fmt.Errorf("field %q of %q has unsupported kind %q", f.Name, gti.Name, f.Type.Kind())
		}
	}

	return doTemplate(w, gti, `
		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid){})
		}
	}

	return nil
}
`)
}

// Generates 'tuple representation' cbor encoders for the given type
func GenMapEncodersForType(gti *GenTypeInfo, flattenEmbeddedStruct bool,
	embeddedByPointerStructs *[]string, w io.Writer) error {
	if flattenEmbeddedStruct {
		if err := emitInitNilEmbeddedStructMethod(w, gti, *embeddedByPointerStructs); err != nil {
			return err
		}
	}

	if err := emitCborMarshalStructMap(w, gti, flattenEmbeddedStruct); err != nil {
		return err
	}

	if err := emitCborUnmarshalStructMap(w, gti, flattenEmbeddedStruct); err != nil {
		return err
	}

	return nil
}
