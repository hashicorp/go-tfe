// Copyright IBM Corp. 2018, 2026

package tfe

import (
	"strings"
	"unicode"

	"github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoft/kiota-abstractions-go/store"
)

// EnableBackingStore shadows NetHttpRequestAdapter.EnableBackingStore, which
// the generated api.NewApiClient calls.
//
// Kiota's own BackingStoreSerializationWriterProxyFactory writes explicit
// nulls for properties set to nil, but it uses the backing store key (the Go
// property name, e.g. "adoOrgName") as the JSON key rather than the
// serialized name ("ado-org-name"). This installs an equivalent proxy that
// writes nulls under the serialized name instead.
func (a *TFERequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {
	if a == nil {
		return
	}

	// Parse node factories are fine as-is. The adapter uses the shared default
	// registry, which this mutates in place.
	parseNodeFactory := a.GetParseNodeFactory()
	if registry, ok := parseNodeFactory.(*serialization.ParseNodeFactoryRegistry); ok {
		registry.Lock()
		for key, value := range registry.ContentTypeAssociatedFactories {
			if _, ok := value.(*store.BackingStoreParseNodeFactory); !ok {
				registry.ContentTypeAssociatedFactories[key] = store.NewBackingStoreParseNodeFactory(value)
			}
		}
		registry.Unlock()
	}

	if registry, ok := a.GetSerializationWriterFactory().(*serialization.SerializationWriterFactoryRegistry); ok {
		registry.Lock()
		for key, value := range registry.ContentTypeAssociatedFactories {
			if _, ok := value.(*nullAwareSerializationWriterFactory); !ok {
				registry.ContentTypeAssociatedFactories[key] = newNullAwareSerializationWriterFactory(value)
			}
		}
		registry.Unlock()
	}

	if factory != nil {
		store.BackingStoreFactoryInstance = factory
	}
}

// nullAwareSerializationWriterFactory mirrors
// store.BackingStoreSerializationWriterProxyFactory, but maps backing store
// keys to serialized property names when writing explicit nulls.
type nullAwareSerializationWriterFactory struct {
	serialization.SerializationWriterFactory
}

func newNullAwareSerializationWriterFactory(factory serialization.SerializationWriterFactory) *nullAwareSerializationWriterFactory {
	return &nullAwareSerializationWriterFactory{
		SerializationWriterFactory: serialization.NewSerializationWriterProxyFactory(
			factory,
			func(p serialization.Parsable) error {
				if bs := backingStoreOf(p); bs != nil {
					bs.SetReturnOnlyChangedValues(true)
				}
				return nil
			},
			func(p serialization.Parsable) error {
				if bs := backingStoreOf(p); bs != nil {
					bs.SetReturnOnlyChangedValues(false)
					bs.SetInitializationCompleted(true)
				}
				return nil
			},
			writeExplicitNulls,
		),
	}
}

func backingStoreOf(p serialization.Parsable) store.BackingStore {
	if m, ok := p.(store.BackedModel); ok {
		return m.GetBackingStore()
	}
	return nil
}

// writeExplicitNulls writes a JSON null for every property that was explicitly
// set to nil. The generated Serialize methods skip nil properties, so these
// are never written twice.
func writeExplicitNulls(p serialization.Parsable, writer serialization.SerializationWriter) error {
	bs := backingStoreOf(p)
	if bs == nil {
		return nil
	}
	keys := bs.EnumerateKeysForValuesChangedToNil()
	if len(keys) == 0 {
		return nil
	}

	names := serializedNames(p)

	for _, key := range keys {
		name, ok := names[normalizeKey(key)]
		if !ok {
			name, ok = names[normalizeKey(strings.TrimSuffix(key, "Escaped"))]
		}
		if !ok {
			// Not a known property (e.g. additional data); leave it out
			// rather than send a key the API doesn't recognize.
			continue
		}
		if err := writer.WriteNullValue(name); err != nil {
			return err
		}
	}
	return nil
}

// serializedNames maps normalized property names to their serialized form,
// using the field deserializers Kiota generates for every model.
func serializedNames(p serialization.Parsable) map[string]string {
	fields := p.GetFieldDeserializers()
	names := make(map[string]string, len(fields))
	for name := range fields {
		names[normalizeKey(name)] = name
	}
	return names
}

// normalizeKey lowercases and strips everything but letters and digits, so
// "adoOrgName", "ado-org-name" and "ado_org_name" all compare equal.
func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
