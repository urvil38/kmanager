package questions

import (
	"errors"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
)

var clusterNameQ = survey.Question{
	Name: "clusterName",
	Prompt: &survey.Input{
		Message: "Enter Cluster name :",
		Help:    "Please provide name of the kubepaas cluster",
	},
	Validate: func(val interface{}) error {
		nameRegex := regexp.MustCompile(`^([^\W])(?:[a-zA-Z1-9-]+)$`)
		if str, ok := val.(string); !ok || !nameRegex.Match([]byte(str)) {
			return errors.New("please enter valid name. Name can contains [ A-Z a-z 1-9 or `-` ]")
		}
		return nil
	},
}

var dnsNameQ = survey.Question{
	Name: "dnsName",
	Prompt: &survey.Input{
		Message: "Enter Domain name :",
		Help:    "Please provide domain name which will be used by kubepaas cluster",
	},
	Validate: func(val interface{}) error {
		nameRegex := regexp.MustCompile(`(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]`)
		if str, ok := val.(string); !ok || !nameRegex.Match([]byte(str)) {
			return errors.New("please enter valid domain name")
		}
		return nil
	},
}

func ProjectPrompt(options []string) []*survey.Question {
	projectPrompt := survey.Question{
		Name: "project",
		Prompt: &survey.Select{
			Message: "Choose google cloud project:",
			Options: options,
		},
	}
	return append([]*survey.Question{}, &projectPrompt)
}

var ClusterName = append([]*survey.Question{}, &clusterNameQ)

var DomainName = append([]*survey.Question{}, &dnsNameQ)
