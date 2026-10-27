package pppoe

import (
	"sort"
)

func ListQueueNames(userID uint, mikrotikID uint64) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	queueNames := make(map[string]struct{})
	for _, command := range []string{"/queue/simple/print", "/queue/tree/print"} {
		reply, err := client.Run(command, "=.proplist=name")
		if err != nil {
			return nil, err
		}
		for _, row := range reply.Re {
			if name := row.Map["name"]; name != "" {
				queueNames[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(queueNames))
	for name := range queueNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
