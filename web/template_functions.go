package web

import (
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/durationutil"
	"github.com/bwmarrin/discordgo"
)

var (
	templateFunctions = map[string]interface{}{
		// Misc
		"lower":            lower,
		"getJoinLink":      getJoinLink,
		"humanizeDuration": func(t time.Time) string { return durationutil.HumanizeDuration(time.Since(t)) },
		// Math
		"add": func(a, b int) int { return a + b },
		// Data types
		"dict":       dict,
		"stringDict": stringDict,
		// Forms content
		"textInput":            textInput,
		"toggleSwitch":         toggleSwitch,
		"numberSelect":         numberSelection,
		"roleOptionsSingle":    roleOptionsSingle,
		"roleOptionsMulti":     roleOptionsMulti,
		"channelOptionsSingle": channelOptionsSingle,
	}
)

func lower(str string) string {
	return strings.ToLower(str)
}

func getJoinLink(guildID interface{}) string {
	joinLink := fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&scope=bot%%20applications.commands+bot&permissions=8&response_type=code&redirect_uri=%s", common.ConfigBotClientID, url.PathEscape(URL+"/dashboard"))
	if guildID != nil {
		joinLink += fmt.Sprintf("&guild_id=%v", guildID)
	}

	return joinLink
}

func dict(pairs ...interface{}) map[int]interface{} {
	result := make(map[int]interface{})
	for i := 0; i < len(pairs); i += 2 {
		key, _ := pairs[i].(int)
		result[key] = pairs[i+1]
	}
	return result
}

func stringDict(pairs ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(pairs); i += 2 {
		key, _ := pairs[i].(string)
		result[key] = pairs[i+1]
	}
	return result
}

// textInput generates a HTML element for a text input field.
//
// Parameters:
//   - currentInput: the current input of the input
//   - uniqueID: unique identifier for the input's ID (used to retrieve and store changed data)
//   - opts: An optional key/value map of additional parameters.
func textInput(currentInput, uniqueID string, opts ...map[string]interface{}) template.HTML {
	var menu strings.Builder

	var leftLable, rightLabel, maxCharacters string
	if len(opts) > 0 {
		leftLable, rightLabel = inputLabel(uniqueID, opts[0])
		if value, ok := opts[0]["maxlength"]; ok {
			maxCharacters = fmt.Sprint(functions.ToInt64(value))
		}
	}

	menu.WriteString(`<div class="input-group mb-3">`)

	menu.WriteString(leftLable)
	menu.WriteString(`<input type="text" class="textInput form-control text-light" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey);" name="` + uniqueID + `" id="` + uniqueID + `" autocomplete="off" value="` + currentInput + `" maxlength="` + maxCharacters + `">`)
	menu.WriteString(rightLabel)

	menu.WriteString("</div>")

	return template.HTML(menu.String())
}

// toggleSwitch generates a HTML object for the custom switch.
// currentState: the bool of the current state of the switch
// uniqueID: string for the input ID (used to retrieve and store changed data)
func toggleSwitch(currentState bool, uniqueID string) template.HTML {
	checked := ""
	if currentState {
		checked = " checked"
	}

	var menu strings.Builder
	menu.WriteString(`<label class="switch">`)
	menu.WriteString(`<input type="checkbox" name="` + uniqueID + `" id="` + uniqueID + `" value="true"` + checked + `/>`)
	menu.WriteString(`<span class="slider" style="left: 5px;"></span>`)
	menu.WriteString(`<span class="knob" style="left: 7px;"></span>`)
	menu.WriteString(`</label>`)

	return template.HTML(menu.String())
}

// numberSelection generates a HTML element for a number input field.
//
// Parameters:
//   - min: lowest number possible
//   - max: highest number possible
//   - currentNumber: the current number
//   - uniqueID: unique identifier for the input's ID (used to retrieve and store changed data)
//   - opts: An optional key/value map of additional parameters such as label settings.
func numberSelection(min, max, currentNumber int64, uniqueID string, opts ...map[string]interface{}) template.HTML {
	var menu strings.Builder
	menu.WriteString(`<div class="input-group mb-3">`)

	var leftLable, rightLabel string
	if len(opts) > 0 {
		leftLable, rightLabel = inputLabel(uniqueID, opts[0])
	}

	menu.WriteString(leftLable)
	if max == 0 {
		menu.WriteString(`<input type="number" name="` + uniqueID + `" id="` + uniqueID + `" min="` + strconv.FormatInt(min, 10) + `" step="1"` + `" class="form-control text-light" placeholder="0" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey);" value="` + strconv.FormatInt(currentNumber, 10) + `">`)
	} else {
		menu.WriteString(`<input type="number" name="` + uniqueID + `" id="` + uniqueID + `" min="` + strconv.FormatInt(min, 10) + `" step="1" max="` + strconv.FormatInt(max, 10) + `" class="form-control text-light" placeholder="0" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey);" value="` + strconv.FormatInt(currentNumber, 10) + `">`)
	}

	menu.WriteString(rightLabel)

	menu.WriteString(`</div>`)

	return template.HTML(menu.String())
}

// roleOptionsSingle generates HTML options for singular role selection
// roles: slice of Discord role objects
// selectedRoleID: string ID of currently selected role
// uniqueID: string for the hidden input ID (used to retrieve and store changed data)
// highestBotRolePosition: the position of the bots highest role
func roleOptionsSingle(roles []*discordgo.Role, selectedRoleID string, uniqueID string, highestBotRolePosition int) template.HTML {
	filteredRoles := make([]*discordgo.Role, 0, len(roles))
	for _, role := range roles {
		if role.Managed || role.Name == "@everyone" {
			continue
		}
		filteredRoles = append(filteredRoles, role)
	}
	sort.Slice(filteredRoles, func(i, j int) bool {
		return filteredRoles[i].Position > filteredRoles[j].Position
	})

	// Button label
	displayText := "Select role"
	if len(selectedRoleID) > 0 {
		label := ""
		for _, role := range filteredRoles {
			if selectedRoleID != role.ID {
				continue
			}
			label = role.Name
			break
		}
		if len(label) > 30 {
			displayText = "1 Selected"
		} else {
			displayText = label
		}
	}

	var menu strings.Builder
	menu.WriteString(`<div class="input-group mb-3">`)
	menu.WriteString(`
		<button class="btn dropdown-toggle text-start flex-grow-1 text-white" type="button" data-bs-toggle="dropdown" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey); border-top-right-radius: var(--bs-btn-border-radius); border-bottom-right-radius: var(--bs-btn-border-radius);">
			<span id="` + uniqueID + `Label">` + template.HTMLEscapeString(displayText) + `</span>
		</button>
		<ul class="dropdown-menu w-100 overflow-auto" style="max-height: 250px;" aria-labelledby="` + uniqueID + `Dropdown">
		<li><a class="dropdown-item dropDownRoleSingleItem" data-value="">None</a></li>
	`)

	for _, role := range filteredRoles {
		disabled := ""
		disabledMsg := ""
		if highestBotRolePosition <= role.Position {
			disabled = " disabled"
			disabledMsg = " (bot higher than role)"
		}

		menu.WriteString(`<li>`)
		menu.WriteString(`<a class="dropdown-item dropDownRoleSingleItem` + disabled + `" data-value="` + role.ID + `">`)
		menu.WriteString(template.HTMLEscapeString(role.Name) + disabledMsg)
		menu.WriteString(`</a></li>`)
	}

	menu.WriteString(`</ul>`)
	menu.WriteString(`<input type="hidden" id="` + uniqueID + `" name="` + uniqueID + `" value="` + template.HTMLEscapeString(selectedRoleID) + `">`)
	menu.WriteString(`</div>`)
	return template.HTML(menu.String())
}

// roleOptionsMulti generates HTML options for multiple role selection
// roles: slice of Discord role objects
// selectedRoleIDs: slice of string IDs of currently selected roles
// uniqueID: string for the input name values (used to retrieve and store changed data)
// highestBotRolePosition: the position of the bots highest role. Use -1 to enable roles above the bot
func roleOptionsMulti(roles []*discordgo.Role, selectedRoleIDs interface{}, uniqueID string, highestBotRolePosition int) template.HTML {
	selectedMap := make(map[string]bool)
	if selectedRoleIDs != nil {
		if roleIDs, ok := selectedRoleIDs.([]string); ok {
			for _, id := range roleIDs {
				selectedMap[id] = true
			}
		}
	}

	filteredRoles := make([]*discordgo.Role, 0, len(roles))
	for _, role := range roles {
		if role.Managed || role.Name == "@everyone" {
			continue
		}
		filteredRoles = append(filteredRoles, role)
	}
	sort.Slice(filteredRoles, func(i, j int) bool {
		return filteredRoles[i].Position > filteredRoles[j].Position
	})

	var selectedNames []string
	for _, role := range filteredRoles {
		if selectedMap[role.ID] {
			selectedNames = append(selectedNames, role.Name)
		}
	}

	// Button label
	displayText := "Select roles"
	if len(selectedNames) > 0 {
		label := strings.Join(selectedNames, ", ")
		if len(selectedNames) > 3 || len(label) > 30 {
			displayText = fmt.Sprintf("%d Selected", len(selectedNames))
		} else {
			displayText = label
		}
	}

	var menu strings.Builder
	menu.WriteString(`<div class="input-group mb-3">`)
	menu.WriteString(`
		<button class="btn dropdown-toggle text-start flex-grow-1 text-white" type="button" data-bs-toggle="dropdown" data-bs-auto-close="outside" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey); border-top-right-radius: var(--bs-btn-border-radius); border-bottom-right-radius: var(--bs-btn-border-radius);">
			<span id="` + uniqueID + `Label">` + template.HTMLEscapeString(displayText) + `</span>
		</button>
		<ul class="dropdown-menu w-100 overflow-auto" style="max-height: 250px;" aria-labelledby="` + uniqueID + `Dropdown">
	`)

	for _, role := range filteredRoles {
		checked := ""
		if selectedMap[role.ID] {
			checked = " checked"
		}
		disabled := ""
		disabledMsg := ""
		if highestBotRolePosition > 0 && (highestBotRolePosition <= role.Position) {
			disabled = " disabled"
			disabledMsg = " (bot higher than role)"
		}

		menu.WriteString(`<li>`)
		menu.WriteString(`<label class="dropdown-item` + disabled + `">`)
		menu.WriteString(`<input type="checkbox" class="dropDownRoleCheckbox me-2" name="` + uniqueID + `" value="` + role.ID + `"` + checked + disabled + `>`)
		menu.WriteString(template.HTMLEscapeString(role.Name) + disabledMsg)
		menu.WriteString(`</label></li>`)
	}
	menu.WriteString(`</ul>`)
	menu.WriteString(`</div>`)
	return template.HTML(menu.String())
}

// channelOptionsSingle generates HTML options for singular channel selection
// channels: slice of Discord channel objects
// selectedChannelID: string ID of currently selected channel
// uniqueID: string for the hidden input ID (used to retrieve and store changed data)
func channelOptionsSingle(channels []*discordgo.Channel, selectedChannelID string, uniqueID string) template.HTML {
	filteredChannels := make([]*discordgo.Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.Type != 0 {
			continue
		}
		filteredChannels = append(filteredChannels, channel)
	}
	sort.Slice(filteredChannels, func(i, j int) bool {
		return filteredChannels[i].Position > filteredChannels[j].Position
	})

	// Button label
	displayText := "Select channel"
	if len(selectedChannelID) > 0 {
		label := ""
		for _, channel := range filteredChannels {
			if selectedChannelID != channel.ID {
				continue
			}
			label = channel.Name
			break
		}
		if len(label) > 30 {
			displayText = "1 Selected"
		} else {
			displayText = label
		}
	}

	var menu strings.Builder
	menu.WriteString(`<div class="input-group mb-3">`)
	menu.WriteString(`
		<button class="btn dropdown-toggle text-start flex-grow-1 text-white" type="button" data-bs-toggle="dropdown" style="background-color: var(--basePurple); border: 1px solid var(--accentGrey); border-top-right-radius: var(--bs-btn-border-radius); border-bottom-right-radius: var(--bs-btn-border-radius);">
			<span id="` + uniqueID + `Label">` + template.HTMLEscapeString(displayText) + `</span>
		</button>
		<ul class="dropdown-menu w-100 overflow-auto" style="max-height: 250px;" aria-labelledby="` + uniqueID + `Dropdown">
		<li><a class="dropdown-item channelListItem" data-value="">None</a></li>
	`)

	for _, channel := range filteredChannels {
		menu.WriteString(`<li>`)
		menu.WriteString(`<a class="dropdown-item channelListItem" data-value="` + channel.ID + `">`)
		menu.WriteString(template.HTMLEscapeString(channel.Name))
		menu.WriteString(`</a></li>`)
	}

	menu.WriteString(`</ul>`)
	menu.WriteString(`<input type="hidden" id="` + uniqueID + `" name="` + uniqueID + `" value="` + template.HTMLEscapeString(selectedChannelID) + `">`)
	menu.WriteString(`</div>`)
	return template.HTML(menu.String())
}

func inputLabel(labelFor string, opts map[string]interface{}) (string, string) {
	var labelEnabled bool
	var labelContent, labelSide string

	if v, ok := opts["label"]; ok {
		if b, ok := v.(bool); ok {
			labelEnabled = b
		}
	}

	if v, ok := opts["labelContent"]; ok {
		if s, ok := v.(string); ok {
			labelContent = s
		}
	}

	if v, ok := opts["labelSide"]; ok {
		if s, ok := v.(string); ok {
			labelSide = s
		}
	}

	if !labelEnabled || (labelSide != "left" && labelSide != "right") {
		return "", ""
	}

	labelContent = convertToImage(labelContent)

	label := fmt.Sprintf("<label for=\"%s\" class=\"input-group-text text-light\" style=\"background-color: var(--primaryTetiaryPurple); border: 1px solid var(--accentGrey)\">%s</label>", labelFor, labelContent)

	if labelSide == "left" {
		return label, ""
	}

	return "", label
}

func convertToImage(emoji string) string {
	re := regexp.MustCompile(`<a?:([a-zA-Z0-9_]+):(\d+)>`)

	return re.ReplaceAllStringFunc(emoji, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		name := matches[1]
		id := matches[2]

		isAnimated := match[1] == 'a'

		format := "png"
		if isAnimated {
			format = "gif"
		}

		url := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", id, format)

		return fmt.Sprintf(`<img src="%s" alt="%s" style="height: 2rem; width: auto;">`, url, name)
	})
}
