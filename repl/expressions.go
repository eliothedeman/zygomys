package zygo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// all Sexp are typed, and have a zero value corresponding to
// the type of the Sexp.

type Sexp interface {
	SexpString() string
	Type() *RegisteredType
}

type SexpPair struct {
	Head Sexp
	Tail Sexp
}

type SexpPointer struct {
	Target        Sexp
	PointedToType *RegisteredType
	MyType        *RegisteredType
}

func NewSexpPointer(pointedTo Sexp, pointedToType *RegisteredType) *SexpPointer {

	ptrRt := GoStructRegistry.GetOrCreatePointerType(pointedToType)
	Q("pointer type is ptrRt = '%#v'", ptrRt)
	p := &SexpPointer{
		Target:        pointedTo,
		PointedToType: pointedToType,
		MyType:        ptrRt,
	}
	return p
}

func (p *SexpPointer) SexpString() string {
	return fmt.Sprintf("%v", p.Target)
	//return fmt.Sprintf("(* %v) %p", p.PointedToType.RegisteredName, p.Target)
}

func (p *SexpPointer) Type() *RegisteredType {
	return p.MyType
}

type SexpInt struct {
	Val int64
	Typ *RegisteredType
}
type SexpBool struct {
	Val bool
	Typ *RegisteredType
}
type SexpFloat struct {
	Val float64
	Typ *RegisteredType
}
type SexpChar struct {
	Val rune
	Typ *RegisteredType
}
type SexpStr struct {
	S        string
	backtick bool
	Typ      *RegisteredType
}

func (r SexpStr) Type() *RegisteredType {
	return GoStructRegistry.Registry["string"]
}

func (r *SexpInt) Type() *RegisteredType {
	return GoStructRegistry.Registry["int64"]
}

func (r SexpFloat) Type() *RegisteredType {
	return GoStructRegistry.Registry["float64"]
}

func (r SexpBool) Type() *RegisteredType {
	return GoStructRegistry.Registry["bool"]
}

func (r SexpChar) Type() *RegisteredType {
	return GoStructRegistry.Registry["int32"]
}

func (r *RegisteredType) Type() *RegisteredType {
	return r
}

type SexpRaw struct {
	Val []byte
	Typ *RegisteredType
}

func (r SexpRaw) Type() *RegisteredType {
	return r.Typ
}

type SexpReflect reflect.Value

func (r SexpReflect) Type() *RegisteredType {
	k := reflectName(reflect.Value(r))
	Q("SexpReflect.Type() looking up type named '%s'", k)
	ty, ok := GoStructRegistry.Registry[k]
	if !ok {
		Q("SexpReflect.Type(): type named '%s' not found", k)
		return nil
	}
	Q("SexpReflect.Type(): type named '%s' found as regtype '%v'", k, ty.SexpString())
	return ty
}

type SexpError struct {
	error
}

func (r SexpError) Type() *RegisteredType {
	return nil // TODO what should this be?
}

type SexpSentinel int

func (r SexpSentinel) Type() *RegisteredType {
	return nil // TODO what should this be?
}

type SexpClosureEnv Scope

func (r SexpClosureEnv) Type() *RegisteredType {
	return nil // TODO what should this be?
}

func (c SexpClosureEnv) SexpString() string {
	scop := Scope(c)
	s, err := scop.Show(scop.env, 0, "")
	if err != nil {
		panic(err)
	}
	return s
}

const (
	SexpNull SexpSentinel = iota
	SexpEnd
	SexpMarker
)

func (sent SexpSentinel) SexpString() string {
	if sent == SexpNull {
		return "nil"
	}
	if sent == SexpEnd {
		return "End"
	}
	if sent == SexpMarker {
		return "Marker"
	}

	return ""
}

func Cons(a Sexp, b Sexp) SexpPair {
	return SexpPair{a, b}
}

func (pair SexpPair) SexpString() string {
	str := "("

	for {
		switch pair.Tail.(type) {
		case SexpPair:
			str += pair.Head.SexpString() + " "
			pair = pair.Tail.(SexpPair)
			continue
		}
		break
	}

	str += pair.Head.SexpString()

	if pair.Tail == SexpNull {
		str += ")"
	} else {
		str += " \\ " + pair.Tail.SexpString() + ")"
	}

	return str
}
func (r SexpPair) Type() *RegisteredType {
	return nil // TODO what should this be?
}

type SexpArray struct {
	Val []Sexp
	Typ *RegisteredType
}

func (r *SexpArray) Type() *RegisteredType {
	if r.Typ == nil {
		if len(r.Val) > 0 {
			// take type from first element
			ty := r.Val[0].Type()
			if ty != nil {
				r.Typ = GoStructRegistry.GetOrCreateSliceType(ty)
			}
		}
	}
	return r.Typ
}

func (arr *SexpArray) SexpString() string {
	if len(arr.Val) == 0 {
		return "[]"
	}

	str := "[" + arr.Val[0].SexpString()
	for _, sexp := range arr.Val[1:] {
		str += " " + sexp.SexpString()
	}
	str += "]"
	return str
}

func (e SexpError) SexpString() string {
	return e.error.Error()
}

type EmbedPath struct {
	ChildName     string
	ChildFieldNum int
}

func GetEmbedPath(e []EmbedPath) string {
	r := ""
	last := len(e) - 1
	for i, s := range e {
		r += s.ChildName
		if i < last {
			r += ":"
		}
	}
	return r
}

type HashFieldDet struct {
	FieldNum     int
	FieldType    reflect.Type
	StructField  reflect.StructField
	FieldName    string
	FieldJsonTag string
	EmbedPath    []EmbedPath // we are embedded if len(EmbedPath) > 0
}
type SexpHash struct {
	TypeName         string
	Map              map[int][]SexpPair
	KeyOrder         []Sexp
	GoStructFactory  *RegisteredType
	NumKeys          int
	GoMethods        []reflect.Method
	GoFields         []reflect.StructField
	GoMethSx         SexpArray
	GoFieldSx        SexpArray
	GoType           reflect.Type
	NumMethod        int
	GoShadowStruct   interface{}
	GoShadowStructVa reflect.Value

	// json tag name -> pointers to example values, as factories for SexpToGoStructs()
	JsonTagMap map[string]*HashFieldDet
	DetOrder   []*HashFieldDet

	// for using these as a scoping model
	DefnEnv    *SexpHash
	SuperClass *SexpHash
	ZMain      SexpFunction
	ZMethods   map[string]*SexpFunction
	env        *Glisp
}

var MethodNotFound = fmt.Errorf("method not found")

func (h *SexpHash) RunZmethod(method string, args []Sexp) (Sexp, error) {
	f, ok := (h.ZMethods)[method]
	if !ok {
		return SexpNull, MethodNotFound
	}

	panic(fmt.Errorf("not done calling %s", f.name))
	//return SexpNull, nil
}

func CallZMethodOnRecordFunction(env *Glisp, name string, args []Sexp) (Sexp, error) {
	narg := len(args)
	if narg < 2 {
		return SexpNull, WrongNargs
	}
	var hash *SexpHash
	switch h := args[0].(type) {
	case *SexpHash:
		hash = h
	default:
		return SexpNull, fmt.Errorf("can only _call on a record")
	}

	method := ""
	switch s := args[1].(type) {
	case SexpSymbol:
		method = s.name
	case SexpStr:
		method = s.S
	default:
		return SexpNull, fmt.Errorf("can only _call with a " +
			"symbol or string as the method name. example: (_call record method:)")
	}

	return hash.RunZmethod(method, args[2:])
}

func (h *SexpHash) SetMain(p *SexpFunction) {
	h.BindSymbol(h.env.MakeSymbol(".main"), p)
}

func (h *SexpHash) SetDefnEnv(p *SexpHash) {
	h.DefnEnv = p
	h.BindSymbol(h.env.MakeSymbol(".parent"), p)
}

func (h *SexpHash) Lookup(env *Glisp, key Sexp) (expr Sexp, err error) {
	return h.HashGet(env, key)
}

func (h *SexpHash) BindSymbol(key SexpSymbol, val Sexp) error {
	return h.HashSet(key, val)
}

func (h *SexpHash) SetGoStructFactory(factory *RegisteredType) {
	h.GoStructFactory = factory
}

var SexpIntSize = 64
var SexpFloatSize = 64

func (r SexpReflect) SexpString() string {
	Q("in SexpReflect.SexpString(); top; type = %T", r)
	if reflect.Value(r).Type().Kind() == reflect.Ptr {
		iface := reflect.Value(r).Interface()
		switch iface.(type) {
		case *string:
			return fmt.Sprintf("`%v`", reflect.Value(r).Elem().Interface())
		default:
			return fmt.Sprintf("%v", reflect.Value(r).Elem().Interface())
		}
	}
	iface := reflect.Value(r).Interface()
	Q("in SexpReflect.SexpString(); type = %T", iface)
	switch iface.(type) {
	default:
		return fmt.Sprintf("%v", iface)
	}
}

func (b SexpBool) SexpString() string {
	if bool(b.Val) {
		return "true"
	}
	return "false"
}

func (i *SexpInt) SexpString() string {
	return strconv.Itoa(int(i.Val))
}

func (f SexpFloat) SexpString() string {
	return strconv.FormatFloat(f.Val, 'g', 5, SexpFloatSize)
}

func (c SexpChar) SexpString() string {
	return "#" + strings.Trim(strconv.QuoteRune(c.Val), "'")
}

func (s SexpStr) SexpString() string {
	if s.backtick {
		return "`" + s.S + "`"
	}
	return strconv.Quote(string(s.S))
}

func (r SexpRaw) SexpString() string {
	return fmt.Sprintf("%#v", []byte(r.Val))
}

type SexpSymbol struct {
	name   string
	number int
	isDot  bool
}

func (sym SexpSymbol) SexpString() string {
	return sym.name
}

func (r SexpSymbol) Type() *RegisteredType {
	return nil // TODO what should this be?
}

func (sym SexpSymbol) Name() string {
	return sym.name
}

func (sym SexpSymbol) Number() int {
	return sym.number
}

type SexpFunction struct {
	name              string
	user              bool
	nargs             int
	varargs           bool
	fun               GlispFunction
	userfun           GlispUserFunction
	orig              Sexp
	closingOverScopes *Closing
	isBuilder         bool // see defbuild; builders are builtins that receive un-evaluated expressions
}

func (sf *SexpFunction) Type() *RegisteredType {
	return nil // TODO what goes here
}

func (sf *SexpFunction) Copy() *SexpFunction {
	cp := *sf
	return &cp
}

func (sf *SexpFunction) SetClosing(clos *Closing) {
	pre, err := sf.ShowClosing(clos.env, 4, "prev")
	panicOn(err)
	newnew, err := sf.ShowClosing(clos.env, 4, "newnew")
	panicOn(err)
	VPrintf("99999 for sfun = %p, in sfun.SetClosing(), prev value is %p = '%s'\n",
		sf, sf.closingOverScopes, pre)
	VPrintf("88888 in sfun.SetClosing(), new  value is %p = '%s'\n", clos, newnew)
	sf.closingOverScopes = clos
}

func (sf *SexpFunction) ShowClosing(env *Glisp, indent int, label string) (string, error) {
	if sf.closingOverScopes == nil {
		return sf.name + " has no captured scopes.", nil
	}
	return sf.closingOverScopes.Show(env, indent, label)
}

func (sf *SexpFunction) ClosingLookupSymbolUntilFunction(sym SexpSymbol) (Sexp, error, *Scope) {
	if sf.closingOverScopes != nil {
		return sf.closingOverScopes.LookupSymbolUntilFunction(sym)
	}
	return SexpNull, SymNotFound, nil
}

func (sf *SexpFunction) ClosingLookupSymbol(sym SexpSymbol) (Sexp, error, *Scope) {
	if sf.closingOverScopes != nil {
		return sf.closingOverScopes.LookupSymbol(sym)
	}
	return SexpNull, SymNotFound, nil
}

func (sf *SexpFunction) SexpString() string {
	if sf.orig == nil {
		return "fn [" + sf.name + "]"
	}
	return sf.orig.SexpString()
}

func IsTruthy(expr Sexp) bool {
	switch e := expr.(type) {
	case SexpBool:
		return e.Val
	case *SexpInt:
		return e.Val != 0
	case SexpChar:
		return e.Val != 0
	case SexpSentinel:
		return e != SexpNull
	}
	return true
}

type SexpStackmark struct {
	sym SexpSymbol
}

func (r SexpStackmark) Type() *RegisteredType {
	return nil // TODO what should this be?
}

func (mark SexpStackmark) SexpString() string {
	return "stackmark " + mark.sym.name
}
