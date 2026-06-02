package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode"
)

// goKeywords is the complete set of Go reserved keywords. Kiota appends
// "Escaped" to any schema name or field name that is a Go keyword so we will
// duplicate that logic here and hope the generated code compiles 🤞.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// SchemaShapeError is returned when a schema with x-helperify does not match
// the expected JSON:API pattern. The message contains hints about why.
type SchemaShapeError struct {
	Message string
}

// Error implements the error interface.
func (e *SchemaShapeError) Error() string {
	return e.Message
}

// SchemaInfo holds everything the generator needs to emit one helper.
// All supported schemas use Pattern B: "type" and "attributes" at the top
// level (flat JSON:API shape). Pattern A (wrapped in "data") schemas should be
// refactored in the upstream OpenAPI spec to use this shape first.
type SchemaInfo struct {
	SchemaName   string // e.g. "team"
	GoHelperBase string // e.g. "Team" — PascalCase, used in helpers pkg
	Collection   bool   // Whether or not the data envelope contains an array of items.

	// Kiota type names referenced in generated code
	KiotaEnvelopeType string // e.g. "TeamRequest"
	KiotaTopType      string // e.g. "Team", "WorkspaceComment"
	KiotaAttrsType    string // e.g. "Team_attributes"

	// Type enum constant used to set body.type, e.g. "TEAMS_TEAM_TYPE"
	TypeEnumConst string

	// The actual JSON:API type string, e.g. "projects"
	TypeEnum string

	Fields []FieldInfo
}

// FieldInfo describes one writable scalar attribute field.
type FieldInfo struct {
	GoField    string // exported Go field name, e.g. "CurrentPassword"
	SetterName string // Kiota setter method name, e.g. "SetCurrentPassword"
	GoType     string // Go type in params struct, e.g. "*string", "string", "*bool"
	Required   bool   // if true GoType is a value type and setter gets &p.Field
}

type parseOptions struct {
}

// parseSchemas inspects components/schemas and returns SchemaInfo for each
// schema that has a x-helperify property and matches the expected JSON:API pattern
func parseSchemas(spec map[string]any) ([]*SchemaInfo, error) {
	components, _ := spec["components"].(map[string]any)
	rawSchemas, _ := components["schemas"].(map[string]any)

	var out []*SchemaInfo
	for name, raw := range rawSchemas {
		s, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		// Schemas must be decorated with the x-helperify extension to be turned
		// into a code-generated helper. It's really only useful for constructing
		// request body data envelope schemas.
		_, ok = s["x-helperify"].(map[string]any)
		if !ok {
			continue
		}

		info, err := parseOneSchema(name, s, rawSchemas, &parseOptions{})
		if err != nil {
			var typeErr *SchemaShapeError
			if errors.As(err, &typeErr) {
				log.Printf("Skipping schema %q: does not match expected JSON:API pattern: %v", name, err)
				continue
			}
			return nil, fmt.Errorf("schema %q: %w", name, err)
		}
		if info != nil {
			out = append(out, info)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].SchemaName < out[j].SchemaName
	})
	return out, nil
}

func resolveRef(namedSchemas, schema map[string]any) (string, map[string]any) {
	if refRaw, hasRef := schema["$ref"]; hasRef {
		key := strings.TrimPrefix(refRaw.(string), "#/components/schemas/")
		resolved, ok := namedSchemas[key]
		if !ok {
			log.Printf("Failed to resolve ref %q", key)
			return "", nil
		}
		return key, resolved.(map[string]any)
	}

	return "", schema
}

// parseOneSchema matches an object with a "type" (enum) and "attributes" at the
// top level.
func parseOneSchema(name string, s map[string]any, namedSchemas map[string]any, options *parseOptions) (*SchemaInfo, error) {
	props, _ := s["properties"].(map[string]any)
	dataRaw, hasData := props["data"].(map[string]any)

	if !hasData {
		return nil, &SchemaShapeError{"'properties' > 'data' missing"} // not a JSON:API request body schema
	}

	dataTypeRaw, hasDataType := dataRaw["type"]

	if !hasDataType {
		return nil, &SchemaShapeError{"'properties' > 'data' > 'type' missing"}
	}

	var resolved map[string]any
	var collection = false
	var topType string
	switch dataTypeRaw.(string) {
	case "array":
		collection = true
		// TODO: dig into the items to find the type value for the enum const
		items, ok := dataRaw["items"].(map[string]any)
		if !ok {
			return nil, &SchemaShapeError{"'properties' > 'data' (type array) > 'items' missing"}
		}
		topType, resolved = resolveRef(namedSchemas, items)
	case "object":
		// Base case
		topType, resolved = resolveRef(namedSchemas, dataRaw)
	}

	if topType == "" {
		topType = name
	}

	log.Printf("topType = %q", topType)

	dataProps, hasDataProps := resolved["properties"].(map[string]any)
	if !hasDataProps {
		return nil, &SchemaShapeError{"inner schema missing 'properties'"} // not a JSON:API request body schema
	}

	attrsRaw, hasAttrs := dataProps["attributes"]
	typeRaw, hasType := dataProps["type"]

	// if options.typeOverride != "" {
	// 	typeRaw = map[string]any{
	// 		"enum": []any{options.typeOverride},
	// 	}
	// 	hasType = true
	// }

	if !hasType || !hasAttrs {
		return nil, &SchemaShapeError{"inner schema properties missing 'type' or 'attributes'"}
	}

	typeValue := firstEnumValue(typeRaw)
	if typeValue == "" {
		return nil, &SchemaShapeError{"inner schema 'type' has no enum values"}
	}

	log.Printf("typeValue = %q", typeValue)

	attrs, _ := attrsRaw.(map[string]any)
	required := stringSet(attrs["required"])
	attrsProps, _ := attrs["properties"].(map[string]any)

	knEnvelope := schemaNameToKiotaType(name)
	knTopType := schemaNameToKiotaType(topType)
	info := SchemaInfo{
		SchemaName:        name,
		GoHelperBase:      schemaToPascalCase(name),
		KiotaEnvelopeType: kiotaTopType(name, knEnvelope),
		KiotaTopType:      kiotaTopType(name, knTopType),

		KiotaAttrsType: knTopType + "_attributes",
		TypeEnumConst:  makeEnumConst(typeValue, knTopType),
		TypeEnum:       typeValue,
		Collection:     collection,
	}
	info.Fields = extractFields(attrsProps, required)
	if len(info.Fields) == 0 {
		return nil, &SchemaShapeError{"inner schema has no writable attributes"}
	}
	return &info, nil
}

// extractFields returns sorted FieldInfo for each writable scalar attribute
// field in the OAS properties map. It skips:
//   - readOnly fields
//   - enum fields (which require Kiota enum types)
//   - complex types (object/array)
//   - date-time format fields (Kiota uses *time.Time, not *string)
func extractFields(props map[string]any, required map[string]bool) []FieldInfo {
	var out []FieldInfo
	for jsonName, rawProp := range props {
		prop, ok := rawProp.(map[string]any)
		if !ok {
			continue
		}
		if ro, _ := prop["readOnly"].(bool); ro {
			continue
		}
		if _, hasEnum := prop["enum"]; hasEnum {
			continue
		}
		oasType, _ := prop["type"].(string)
		if oasType == "object" || oasType == "array" || oasType == "" {
			continue
		}
		// date-time fields: Kiota generates *time.Time setters, not *string.
		if fmt_, _ := prop["format"].(string); fmt_ == "date-time" || fmt_ == "date" || fmt_ == "time" {
			continue
		}
		isReq := required[jsonName]
		gt := oasTypeToGo(oasType, isReq)
		if gt == "" {
			continue
		}
		goField := jsonNameToGoField(jsonName)
		out = append(out, FieldInfo{
			GoField:    goField,
			SetterName: "Set" + goField,
			GoType:     gt,
			Required:   isReq,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GoField < out[j].GoField
	})
	return out
}

// oasTypeToGo maps an OAS scalar type to a Go type string.
// Required fields use value types; optional fields use pointer types.
func oasTypeToGo(oasType string, required bool) string {
	var base string
	switch oasType {
	case "string":
		base = "string"
	case "boolean":
		base = "bool"
	case "integer":
		base = "int32"
	case "number":
		base = "float64"
	default:
		return ""
	}
	if required {
		return base
	}
	return "*" + base
}

// jsonNameToGoField converts a JSON field name (kebab or snake case) to an
// exported Go identifier using PascalCase. If the original field name is a Go
// keyword, "Escaped" is appended to match Kiota's naming (e.g. "type" → "TypeEscaped").
func jsonNameToGoField(name string) string {
	result := toPascalCase(splitOnSeps(name))
	if goKeywords[name] {
		result += "Escaped"
	}
	return result
}

// kiotaTopType returns the Kiota top-level struct name for a schema. This differs
// from the base name only when the entire schema name is a Go keyword (e.g. "var"
// → base "Var" but top type "VarEscaped"). Sub-types (attributes, enum) always
// use the unescaped base name.
func kiotaTopType(schemaName, kiotaBaseName string) string {
	if goKeywords[schemaName] {
		return kiotaBaseName + "Escaped"
	}
	return kiotaBaseName
}

// schemaNameToKiotaType converts a schema name to the Kiota-generated Go struct name.
//
// Kiota's naming rules:
//   - Hyphen-separated names ("plan-export", "workspace-comment") → PascalCase ("PlanExport")
//   - Underscore-separated names ("account_password") → capitalize first word only ("Account_password")
//   - Single words ("team", "run") → capitalize ("Team", "Run")
func schemaNameToKiotaType(name string) string {
	if strings.ContainsRune(name, '-') {
		// Hyphen names: full PascalCase
		return schemaToPascalCase(name)
	}
	// Underscore names: only the first word is capitalized
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return capitalize(name)
	}
	parts[0] = capitalize(parts[0])
	return strings.Join(parts, "_")
}

// schemaToPascalCase converts any schema name (hyphen or underscore separated)
// to PascalCase. Used for the helpers package names (GoHelperBase).
func schemaToPascalCase(name string) string {
	return toPascalCase(splitOnSeps(name))
}

func splitOnSeps(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
}

func toPascalCase(words []string) string {
	var b strings.Builder
	for _, w := range words {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// makeEnumConst derives the Kiota enum constant name from the JSON:API type
// value and the Kiota type name.
//
// Pattern: {UPPERCASE_TYPEVAL}_{UPPERCASE_KIOTANAME}_TYPE
//
// Examples:
//
//	("users",        "Account_password_data") → "USERS_ACCOUNT_PASSWORD_DATA_TYPE"
//	("teams",        "Team")                  → "TEAMS_TEAM_TYPE"
//	("plan-exports", "PlanExport")            → "PLANEXPORTS_PLANEXPORT_TYPE"
//	("run-triggers", "RunTrigger")            → "RUNTRIGGERS_RUNTRIGGER_TYPE"
func makeEnumConst(typeValue, kiotaTypeName string) string {
	tv := strings.ToUpper(strings.ReplaceAll(typeValue, "-", ""))
	kn := strings.ToUpper(kiotaTypeName)
	return tv + "_" + kn + "_TYPE"
}

func firstEnumValue(raw any) string {
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	vals, _ := m["enum"].([]any)
	if len(vals) == 0 {
		return ""
	}
	s, _ := vals[0].(string)
	return s
}

func stringSet(raw any) map[string]bool {
	arr, _ := raw.([]any)
	set := make(map[string]bool, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			set[s] = true
		}
	}
	return set
}
