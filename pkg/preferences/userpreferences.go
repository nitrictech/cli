// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package preferences

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"

	"github.com/nitrictech/cli/pkg/utils"
)

const feedbackMsg = "Thanks for trying nitric!\nIf you have feedback you can raise issues on GitHub https://github.com/nitrictech/nitric or come talk with us directly on Discord discord.com/invite/Webemece5C"

var (
	defaultBackendPassPhrase = uuid.NewString()
	defaultPreferences       = &UserPreferences{
		Feedback: FeedbackPreferences{
			AskFeedback: true,
			LastPrompt:  time.Now().AddDate(-1, 0, 0).Format(time.RFC822),
		},
	}
)

type UserPreferences struct {
	Feedback               FeedbackPreferences `json:"feedback"`
	LocalBackendPassPhrase string
}

type FeedbackPreferences struct {
	AskFeedback bool   `json:"askFeedback"`
	LastPrompt  string `json:"lastPrompt"`
}

func GetLocalPassPhraseFile() (string, error) {
	if _, err := os.Stat(utils.NitricLocalPassphrasePath()); errors.Is(err, os.ErrNotExist) {
		err = os.WriteFile(utils.NitricLocalPassphrasePath(), []byte(defaultBackendPassPhrase), os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	return utils.NitricLocalPassphrasePath(), nil
}

func ReadUserPreferences() (*UserPreferences, error) {
	// If there are no preferences, set as default
	if _, err := os.Stat(utils.NitricPreferencesPath()); errors.Is(err, os.ErrNotExist) {
		err := defaultPreferences.WriteToFile()
		if err != nil {
			return nil, err
		}
	}

	contents, err := os.ReadFile(utils.NitricPreferencesPath())
	if err != nil {
		return nil, err
	}

	var up *UserPreferences

	err = json.Unmarshal(contents, &up)
	if err != nil {
		return nil, err
	}

	return up, nil
}

func (up *UserPreferences) WriteToFile() error {
	file, err := os.Create(utils.NitricPreferencesPath())
	if err != nil {
		return err
	}
	defer file.Close()

	contents, err := json.Marshal(up)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(contents))
	if err != nil {
		return err
	}

	return nil
}

func (f *FeedbackPreferences) hasBeenWeek() bool {
	weekAgo := time.Now().AddDate(0, 0, -7)

	lastPrompt, err := time.Parse(time.RFC822, f.LastPrompt)
	if err != nil {
		return false
	}

	return lastPrompt.Before(weekAgo)
}

func PromptFeedback() error {
	up, err := ReadUserPreferences()
	if err != nil {
		return err
	}

	if up.Feedback.AskFeedback && up.Feedback.hasBeenWeek() {
		up.Feedback.LastPrompt = time.Now().Format(time.RFC822)

		fmt.Println(feedbackMsg)

		feedbackResp := struct{ FeedbackName string }{}

		err := survey.Ask([]*survey.Question{{
			Name: "feedbackName",
			Prompt: &survey.Select{
				Message: "Ask again later?",
				Options: []string{"Yes", "No"},
				Default: "No",
			},
		}}, &feedbackResp)
		if err != nil {
			return err
		}

		if feedbackResp.FeedbackName == "No" {
			up.Feedback.AskFeedback = false
		}

		err = up.WriteToFile()
		if err != nil {
			return err
		}
	}

	return nil
}
