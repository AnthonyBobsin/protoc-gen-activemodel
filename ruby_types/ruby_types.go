package ruby_types

import (
	"fmt"
	"log"
	"strings"

	pgs "github.com/lyft/protoc-gen-star"
)

type methodType int

const (
	methodTypeGetter methodType = iota
	methodTypeSetter
	methodTypeInitializer
)

// intersection between pgs.FieldType and pgs.FieldTypeElem
type FieldType interface {
	ProtoType() pgs.ProtoType
	IsEmbed() bool
	IsEnum() bool
	Imports() []pgs.File
	Enum() pgs.Enum
	Embed() pgs.Message
}

// intersection between pgs.Message and pgs.Enum
type EntityWithParent interface {
	pgs.Entity
	Parent() pgs.ParentEntity
}

func RubyModules(file pgs.File) []string {
	p := RubyPackage(file)
	split := strings.Split(p, "::")
	modules := make([]string, 0)
	for i := 0; i < len(split); i++ {
		modules = append(modules, strings.Join(split[0:i+1], "::"))
	}
	return modules
}

func RubyPackage(file pgs.File) string {
	pkg := file.Descriptor().GetOptions().GetRubyPackage()
	if pkg == "" {
		pkg = file.Descriptor().GetPackage()
	}
	pkg = strings.Replace(pkg, ".", "::", -1)
	// right now the ruby_out doesn't camelcase the ruby_package, but this results in invalid classes, so do it:
	return pgs.Name(pkg).UpperCamelCase().String()
}

func RubyMessageType(entity EntityWithParent) string {
	names := make([]string, 0)
	outer := entity
	ok := true
	for ok {
		name := outer.Name().String()
		names = append([]string{strings.Title(name)}, names...)
		outer, ok = outer.Parent().(pgs.Message)
	}
	return fmt.Sprintf("%s::%s", RubyPackage(entity.File()), strings.Join(names, "::"))
}

func RubyGetterFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeGetter)
}

func RubySetterFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeSetter)
}

func RubyInitializerFieldType(field pgs.Field) string {
	return rubyFieldType(field, methodTypeInitializer)
}

func rubyFieldType(field pgs.Field, mt methodType) string {
	var rubyType string

	t := field.Type()

	if t.IsMap() {
		rubyType = rubyFieldMapType(field, t, mt)
	} else if t.IsRepeated() {
		rubyType = rubyFieldRepeatedType(field, t, mt)
	} else {
		rubyType = rubyProtoTypeElem(field, t)
	}

	return rubyType
}

func rubyFieldMapType(field pgs.Field, ft pgs.FieldType, mt methodType) string {
	// TODO(bobsin): handle this properly
	if mt == methodTypeSetter {
		return "Google::Protobuf::Map"
	}
	key := rubyProtoTypeElem(field, ft.Key())
	value := rubyProtoTypeElem(field, ft.Element())
	return fmt.Sprintf("T::Hash[%s, %s]", key, value)
}

func rubyFieldRepeatedType(field pgs.Field, ft pgs.FieldType, mt methodType) string {
	// TODO(bobsin): handle this properly
	// An enumerable/array is not accepted at the setter
	// See: https://github.com/protocolbuffers/protobuf/issues/4969
	// See: https://developers.google.com/protocol-buffers/docs/reference/ruby-generated#repeated-fields
	if mt == methodTypeSetter {
		return "Google::Protobuf::RepeatedField"
	}
	value := rubyProtoTypeElem(field, ft.Element())
	return fmt.Sprintf("T::Array[%s]", value)
}

func rubyProtoTypeElem(field pgs.Field, ft FieldType) string {
	pt := ft.ProtoType()
	if pt.IsInt() {
		return ":integer"
	}
	if pt.IsNumeric() {
		return ":float"
	}
	if pt == pgs.StringT || pt == pgs.BytesT {
		return ":string"
	}
	if pt == pgs.BoolT {
		return ":boolean"
	}
	if pt == pgs.EnumT {
		return ":string"
	}
	if pt == pgs.MessageT {
		return RubyMessageType(ft.Embed())
	}
	log.Panicf("Unsupported field type for field: %v\n", field.Name().String())
	return ""
}
