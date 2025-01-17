// Copyright 2022 Tigris Data, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package update

import (
	"fmt"
	"testing"

	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	"github.com/tigrisdata/tigris/lib/json"
)

func TestMergeAndGet(t *testing.T) {
	cases := []struct {
		inputDoc    jsoniter.RawMessage
		existingDoc jsoniter.RawMessage
		outputDoc   jsoniter.RawMessage
		apply       FieldOPType
	}{
		{
			[]byte(`{"a": 10}`),
			[]byte(`{"a": 1, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			[]byte(`{"a": 10, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			Set,
		}, {
			[]byte(`{"b": "bar", "a": 10}`),
			[]byte(`{"a": 1, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			[]byte(`{"a": 10, "b": "bar", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			Set,
		}, {
			[]byte(`{"b": "test", "c": 10.22}`),
			[]byte(`{"a": 1, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			[]byte(`{"a": 1, "b": "test", "c": 10.22, "d": {"f": 22, "g": 44}}`),
			Set,
		}, {
			[]byte(`{"c": 10.000022, "e": "new"}`),
			[]byte(`{"a": 1, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			[]byte(`{"a": 1, "b": "foo", "c": 10.000022, "d": {"f": 22, "g": 44},"e":"new"}`),
			Set,
		}, {
			[]byte(`{"e": "again", "a": 1.000000022, "c": 23}`),
			[]byte(`{"a": 1, "b": "foo", "c": 1.01, "d": {"f": 22, "g": 44}}`),
			[]byte(`{"a": 1.000000022, "b": "foo", "c": 23, "d": {"f": 22, "g": 44},"e":"again"}`),
			Set,
		}, {
			[]byte(`{"e": "again", "d.f": 29, "d.g": "bar", "d.h": "new nested"}`),
			[]byte(`{"a":1, "b":"foo", "c":1.01, "d": {"f": 22, "g": "foo"}}`),
			[]byte(`{"a":1, "b":"foo", "c":1.01, "d": {"f": 29, "g": "bar","h":"new nested"},"e":"again"}`),
			Set,
		},
	}
	for _, c := range cases {
		reqInput := []byte(fmt.Sprintf(`{"%s": %s}`, c.apply, c.inputDoc))
		f, err := BuildFieldOperators(reqInput)
		require.NoError(t, err)

		actualOut, err := f.MergeAndGet(c.existingDoc)
		require.NoError(t, err)
		require.Equal(t, c.outputDoc, actualOut, fmt.Sprintf("exp '%s' actual '%s'", string(c.outputDoc), string(actualOut)))
	}
}

func TestMergeAndGetWithUnset(t *testing.T) {
	cases := []struct {
		inputSet    jsoniter.RawMessage
		inputRemove jsoniter.RawMessage
		existingDoc jsoniter.RawMessage
		outputDoc   jsoniter.RawMessage
	}{
		{
			[]byte(`{"a":10}`),
			[]byte(`["a", "nested"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g":44}}`),
			[]byte(`{"b":"first","c":1.01}`),
		}, {
			[]byte(`{"b":"second","a":10}`),
			[]byte(`["c"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g":44}}`),
			[]byte(`{"a":10,"b":"second","nested":{"f":22,"g":44}}`),
		}, {
			[]byte(`{"b":"second","c":10.22}`),
			[]byte(`["nested.f"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g":44}}`),
			[]byte(`{"a":1,"b":"second","c":10.22,"nested":{"g":44}}`),
		}, {
			[]byte(`{"c":10.000022,"e":"not_present"}`),
			[]byte(`["nested.f", "nested.g"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g": 4}}`),
			[]byte(`{"a":1,"b":"first","c":10.000022,"nested":{},"e":"not_present"}`),
		}, {
			[]byte(`{"e":"not_present","a":1.000000022,"c":23}`),
			[]byte(`["c", "b"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g":44}}`),
			[]byte(`{"a":1.000000022,"nested":{"f":22,"g":44},"e":"not_present"}`),
		}, {
			[]byte(`{"e":"not_present","nested.f":29,"nested.g":"bar","nested.h":"new nested"}`),
			[]byte(`["z", "y"]`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":22,"g":"foo"}}`),
			[]byte(`{"a":1,"b":"first","c":1.01,"nested":{"f":29,"g":"bar","h":"new nested"},"e":"not_present"}`),
		},
	}
	for _, c := range cases {
		reqInput := []byte(fmt.Sprintf(`{"$set": %s, "$unset": %s}`, c.inputSet, c.inputRemove))
		f, err := BuildFieldOperators(reqInput)
		require.NoError(t, err)

		actualOut, err := f.MergeAndGet(c.existingDoc)
		require.NoError(t, err)
		require.Equal(t, c.outputDoc, actualOut, fmt.Sprintf("exp '%s' actual '%s'", string(c.outputDoc), string(actualOut)))
	}
}

func TestMergeAndGet_MarshalInput(t *testing.T) {
	cases := []struct {
		inputDoc    map[string]interface{}
		existingDoc map[string]interface{}
		outputDoc   jsoniter.RawMessage
		apply       FieldOPType
	}{
		{
			map[string]interface{}{
				"int_value":    200,
				"string_value": "simple_insert1_update_modified",
				"bool_value":   false,
				"double_value": 200.00001,
				"bytes_value":  []byte(`"simple_insert1_update_modified"`),
			},
			map[string]interface{}{
				"pkey_int":     100,
				"int_value":    100,
				"string_value": "simple_insert1_update",
				"bool_value":   true,
				"double_value": 100.00001,
				"bytes_value":  []byte(`"simple_insert1_update"`),
			},
			[]byte(`{"pkey_int":100,"int_value":200,"string_value":"simple_insert1_update_modified","bool_value":false,"double_value":200.00001,"bytes_value":"InNpbXBsZV9pbnNlcnQxX3VwZGF0ZV9tb2RpZmllZCI="}`),
			Set,
		},
	}
	for _, c := range cases {
		reqInput := make(map[string]interface{})
		reqInput[string(c.apply)] = c.inputDoc
		input, err := jsoniter.Marshal(reqInput)
		require.NoError(t, err)
		f, err := BuildFieldOperators(input)
		require.NoError(t, err)
		existingDoc, err := jsoniter.Marshal(c.existingDoc)
		require.NoError(t, err)
		actualOut, err := f.MergeAndGet(existingDoc)
		require.NoError(t, err)
		require.JSONEqf(t, string(c.outputDoc), string(actualOut), fmt.Sprintf("exp '%s' actual '%s'", string(c.outputDoc), string(actualOut)))
	}
}

func BenchmarkSetNoDeserialization(b *testing.B) {
	existingDoc := []byte(`{
	"name": "Women's Fiona Handbag",
	"brand": "Michael Cors",
	"labels": "Handbag, Purse, Women's fashion",
	"price": 99999.12345,
	"key": "1",
	"categories": ["random", "fashion", "handbags", "women's"],
	"description": "A typical product catalog will have many json objects like this. This benchmark is testing if not deserializing is better than deserializing JSON inputs and existing doc.",
	"random": "abc defg hij klm nopqr stuv wxyz 1234 56 78 90 abcd efghijkl mnopqrstuvwxyzA BCD EFGHIJKL MNOPQRS TUVW XYZ"
}`)

	f, err := BuildFieldOperators([]byte(`{"$set": {"b": "bar", "a": 10}}`))
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		err = f.testSetNoDeserialization(existingDoc, []byte(`{"$set": {"name": "Men's Wallet", "labels": "Handbag, Purse, Men's fashion, shoes, clothes", "price": 75}}`))
		require.NoError(b, err)
	}
}

func BenchmarkSetDeserializeInput(b *testing.B) {
	existingDoc := []byte(`{
	"name": "Women's Fiona Handbag",
	"brand": "Michael Cors",
	"labels": "Handbag, Purse, Women's fashion",
	"price": 99999.12345,
	"key": "1",
	"categories": ["random", "fashion", "handbags", "women's"],
	"description": "A typical product catalog will have many json objects like this. This benchmark is testing if deserializing is better than not deserializing JSON inputs and existing doc.",
	"random": "abc defg hij klm nopqr stuv wxyz 1234 56 78 90 abcd efghijkl mnopqrstuvwxyzA BCD EFGHIJKL MNOPQRS TUVW XYZ"
}`)

	f, err := BuildFieldOperators([]byte(`{"$set": {"b": "bar", "a": 10}}`))
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		mp, err := json.Decode(existingDoc)
		require.NoError(b, err)

		err = f.testSetDeserializeInput(mp, []byte(`{"$set": {"name": "Men's Wallet", "labels": "Handbag, Purse, Men's fashion, shoes, clothes", "price": 75}}`))
		require.NoError(b, err)
	}
}

func (factory *FieldOperatorFactory) testSetDeserializeInput(outMap map[string]any, setDoc jsoniter.RawMessage) error {
	setMap, err := json.Decode(setDoc)
	if err != nil {
		return err
	}

	for key, value := range setMap {
		outMap[key] = value
	}

	return nil
}

func (factory *FieldOperatorFactory) testSetNoDeserialization(input jsoniter.RawMessage, setDoc jsoniter.RawMessage) error {
	var (
		output []byte = input
		err    error
	)
	err = jsonparser.ObjectEach(setDoc, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if dataType == jsonparser.String {
			value = []byte(fmt.Sprintf(`"%s"`, value))
		}
		output, err = jsonparser.Set(output, value, string(key))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
