package infobot

import (
	"testing"

	"github.com/golang/protobuf/proto"
	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/infobot/v1"
	"github.com/stretchr/testify/require"
)

func TestFactStore_LoadFactPack(t *testing.T) {
	fs := MakeFactStore()

	err := fs.LoadFactPack("examples/facts.txt")
	require.NoError(t, err)

	require.Equal(t, "roses are red", Output(fs.GetFact("roses")))
	require.Equal(t, "violets are blue", Output(fs.GetFact("violets")))
	require.Equal(t, "one is fish", Output(fs.GetFact("one")))
	require.Equal(t, "red fish is blue fish", Output(fs.GetFact("red fish")))
	require.Nil(t, fs.GetFact("elephant"))
	require.Equal(t, "monkey is a banana eater => yum", Output(fs.GetFact("monkey")))
}

func TestFactStore_SetFact(t *testing.T) {
	fs := MakeFactStore()

	fs.SetFact(&v1.Fact{
		Name:     "test fact",
		Value:    "these are fact details",
		IsPlural: false,
	})
	require.NotNil(t, fs.GetFact("test fact"))
	require.Equal(t, "test fact is these are fact details", Output(fs.GetFact("test fact")))
}

func TestFactStore_DeleteFact(t *testing.T) {
	fs := MakeFactStore()

	fs.SetFact(&v1.Fact{
		Name:     "42",
		Value:    "life, universe, and everything",
		IsPlural: false,
	})
	require.NotNil(t, fs.GetFact("42"))
	require.Equal(t, "42 is life, universe, and everything", Output(fs.GetFact("42")))

	fs.DeleteFact("42")
	require.Nil(t, fs.GetFact("test fact"))
}

func TestFactStore_HumanFactSet(t *testing.T) {
	fs := MakeFactStore()

	fs.HumanFactSet("roses are red")
	require.Equal(t, "roses are red", Output(fs.GetFact("roses")))

	fs.HumanFactSet("the quick brown fox is jumping over the lazy dog")
	require.NotNil(t, fs.GetFact("the quick brown fox"))
	require.Equal(t, "the quick brown fox is jumping over the lazy dog", Output(fs.GetFact("the quick brown fox")))

	fs.HumanFactSet("42 is the answer to life, the universe, and everything")
	require.NotNil(t, fs.GetFact("42"))
	require.Equal(t, "42 is the answer to life, the universe, and everything", Output(fs.GetFact("42")))

	fs.HumanFactSet("monkeys are animals that live in trees and are animals with tails")
	require.NotNil(t, fs.GetFact("monkeys"))
	require.Equal(t, "monkeys are animals that live in trees and are animals with tails", Output(fs.GetFact("monkeys")))
}

func TestFactStore_HumanForgetFact(t *testing.T) {
	fs := MakeFactStore()

	fs.HumanFactSet("roses are red")
	require.NotNil(t, fs.GetFact("roses"))
	require.Equal(t, "roses are red", Output(fs.GetFact("roses")))

	fs.HumanFactForget("forget roses")
	require.Nil(t, fs.GetFact("roses"))
}

func TestFactStore_Serialize(t *testing.T) {
	fs := MakeFactStore()

	fs.HumanFactSet("roses are red")
	require.NotNil(t, fs.GetFact("roses"))
	require.Equal(t, "roses are red", Output(fs.GetFact("roses")))

	out, err := fs.Serialize()
	require.NoError(t, err)

	newFactstore := &v1.FactStore{}
	err = proto.Unmarshal(out, newFactstore)
	require.NoError(t, err)

	fs.factStore = newFactstore

	require.NotNil(t, fs.GetFact("roses"))
	require.Equal(t, "roses are red", Output(fs.GetFact("roses")))
}
