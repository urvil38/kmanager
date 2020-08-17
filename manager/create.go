package manager

import (
	"context"
	"log"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urvil38/kmanager/questions"
)

func (m *Manager) Create() error {

	if err := survey.Ask(questions.ClusterName, &m.Cc.Name); err != nil {
		return err
	}

	if err := survey.Ask(questions.DomainName, &m.Cc.DNSName); err != nil {
		return err
	}

	gCmds, err := m.Cc.initGCloudCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	for _, cmd := range gCmds.cmds {
		if !cmd.internal {
			cmd.execute(context.Background(), m.Cc)
			if !cmd.succeed {
				// fmt.Println(cmd.stderr)
				continue
			}
		}
	}

	_, err = m.Cc.initKubeCmdSet()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
