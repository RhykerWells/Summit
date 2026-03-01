package dcommand

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// helper to create empty Data used by parser; only Author is used for embed building
func newTestData() *Data {
	return &Data{Author: &discordgo.User{Username: "tester"}}
}

func TestHandleMissingOrInvalidArgs_OptionalArgs(t *testing.T) {
	cmd := SummitCommand{
		Command:      "foo",
		Args:         []*Arg{{Name: "one", Type: String}, {Name: "two", Type: String}},
		ArgsRequired: 0, // no required args -> everything optional
	}
	data := newTestData()
	err, embed := handleMissingOrInvalidArgs(cmd, data, []string{})
	if err != nil {
		t.Fatalf("did not expect error when invoking with zero tokens, got %v", err)
	}
	if embed != nil {
		t.Fatalf("expected no embed message, got %+v", embed)
	}
	if len(data.ParsedArgs) != len(cmd.Args) {
		t.Fatalf("expected %d parsed args, got %d", len(cmd.Args), len(data.ParsedArgs))
	}
	for i, p := range data.ParsedArgs {
		if p.Value != nil {
			t.Errorf("expected nil value for arg %d but got %v", i, p.Value)
		}
	}
}

func TestHandleMissingOrInvalidArgs_RequiredArgs(t *testing.T) {
	cmd := SummitCommand{
		Command:      "bar",
		Args:         []*Arg{{Name: "a", Type: String}, {Name: "b", Type: String}},
		ArgsRequired: 1,
	}

	err, embed := handleMissingOrInvalidArgs(cmd, newTestData(), []string{})
	if err == nil {
		t.Fatal("expected error for missing required argument")
	}
	if embed == nil {
		t.Fatal("expected embed")
	}
	if strings.Contains(embed.Description, "[a:") {
		t.Errorf("did not expect optional formatting: %s", embed.Description)
	}
}

func TestHandleMissingOrInvalidArgs_ComboFormatting(t *testing.T) {
	cmd := SummitCommand{
		Command:        "baz",
		Args:           []*Arg{{Name: "x", Type: String}, {Name: "y", Type: String}},
		ArgumentCombos: [][]int{{0}, {1}},
	}
	err, embed := handleMissingOrInvalidArgs(cmd, newTestData(), []string{"foo", "bar"})
	if err != nil {
		t.Fatalf("unexpected error for valid combo: %v", err)
	}
	if embed != nil {
		t.Fatalf("expected no embed for valid combo, got %+v", embed)
	}
}

// ensure the complex mute combos behave correctly when the reason has spaces
func TestHandleMissingOrInvalidArgs_MuteCombos(t *testing.T) {
	cmd := SummitCommand{
		Command:      "mute",
		ArgsRequired: 3,
		// use plain strings for testing to avoid discord lookups
		Args: []*Arg{
			{Name: "Member", Type: String},
			{Name: "Duration", Type: Duration},
			{Name: "Reason", Type: String},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2}, // member duration reason
			{0, 2, 1}, // member reason duration
			{0, 1},    // member duration only
			{0, 2},    // member reason only
			{0},       // member only
		},
	}

	// order: member reason duration; reason has spaces
	tokens := []string{"1095320115211423744", "blsah", "blah", "blah", "2d"}
	parsed := newTestData()
	err, embed := handleMissingOrInvalidArgs(cmd, parsed, tokens)
	if err != nil {
		t.Fatalf("expected no error for valid mute combo, got %v", err)
	}
	if embed != nil {
		t.Fatalf("expected no embed, got %+v", embed)
	}

	// verify parsed args: member id, duration 2d, reason = join of tokens[1:len-1]
	if parsed.ParsedArgs[0].String() != "1095320115211423744" {
		t.Errorf("member parsed incorrectly: %s", parsed.ParsedArgs[0].String())
	}
	if parsed.ParsedArgs[1].String() != "2d" {
		t.Errorf("duration parsed incorrectly: %s", parsed.ParsedArgs[1].String())
	}
	if parsed.ParsedArgs[2].String() != "blsah blah blah" {
		t.Errorf("reason parsed incorrectly, got %s", parsed.ParsedArgs[2].String())
	}

	// also test member duration reason order with extra words in reason
	tokens = []string{"1095", "1d", "some", "multi", "word", "reason"}
	parsed = newTestData()
	err, _ = handleMissingOrInvalidArgs(cmd, parsed, tokens)
	if err != nil {
		t.Fatalf("unexpected error parsing order member duration reason: %v", err)
	}
	if parsed.ParsedArgs[1].String() != "1d" {
		t.Errorf("duration not parsed correctly in first-order combo: %s", parsed.ParsedArgs[1].String())
	}
	if parsed.ParsedArgs[2].String() != "some multi word reason" {
		t.Errorf("reason not absorbed correctly: %s", parsed.ParsedArgs[2].String())
	}
}
