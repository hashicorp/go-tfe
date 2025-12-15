// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

var validResourceName = regexp.MustCompile(`^[a-zA-Z_]+$`).MatchString

func generateResourceTemplate(name string) ResourceTemplate {
	var pluralName string
	pluralize := pluralize.NewClient()

	if pluralize.IsPlural(name) {
		pluralName = name
		name = pluralize.Singular(name)
	} else {
		pluralName = pluralize.Plural(name)
	}

	camelName := strcase.ToCamel(name)

	return ResourceTemplate{
		PrimaryTag:        strings.ReplaceAll(name, "_", "-"),
		Name:              strings.ReplaceAll(name, "_", " "),
		PluralName:        strings.ReplaceAll(pluralName, "_", " "),
		Resource:          camelName,
		ResourceInterface: strcase.ToCamel(pluralName),
		ResourceStruct:    strcase.ToLowerCamel(pluralName),
		ResourceID:        fmt.Sprintf("%sID", camelName),
		ListOptions:       fmt.Sprintf("%sListOptions", camelName),
		ReadOptions:       fmt.Sprintf("%sReadOptions", camelName),
		CreateOptions:     fmt.Sprintf("%sCreateOptions", camelName),
		UpdateOptions:     fmt.Sprintf("%sUpdateOptions", camelName),
	}
}

func main() {
	var resourceName string

	if len(os.Args) < 2 {
		log.Fatal("usage: <resource name>")
	}

	if os.Args[1] == "-h" {
		fmt.Println(helpTemplate)
		return
	} else {
		resourceName = strings.ToLower(os.Args[1])
	}

	if !validResourceName(resourceName) {
		log.Fatal("resource name can only contain letters or underscores.")
	}

	resourceTmpl := generateResourceTemplate(resourceName)

	tmp, err := template.New("source").Parse(sourceTemplate)
	if err != nil {
		log.Fatal(err)
	}

	sourceFile, err := os.Create("../../" + resourceName + ".go")
	if err != nil {
		log.Fatal(err)
	}

	defer sourceFile.Close()

	fmt.Printf("Generating %s.go\n", resourceName)
	err = tmp.Execute(sourceFile, resourceTmpl)
	if err != nil {
		log.Fatal(err)
	}

	tmp, err = template.New("source").Parse(testTemplate)
	if err != nil {
		log.Fatal(err)
	}

	testFile, err := os.Create("../../" + resourceName + "_integration_test.go")
	if err != nil {
		log.Fatal(err)
	}

	defer testFile.Close()

	fmt.Printf("Generating %s_integration_test.go\n", resourceName)
	err = tmp.Execute(testFile, resourceTmpl)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done generating files for new resource: %s\n", resourceTmpl.Resource)
}
