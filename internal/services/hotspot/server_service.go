package hotspot

const serverPath = "/ip/hotspot"

// defaultServer adalah nilai server bawaan user hotspot (berlaku di semua server).
const defaultServer = "all"

// ListServerNames mengembalikan nama server hotspot (/ip/hotspot) untuk pilihan field "server".
func ListServerNames(userID uint, mikrotikID uint64) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(serverPath+"/print", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(reply.Re))
	for _, row := range reply.Re {
		if name := row.Map["name"]; name != "" {
			names = append(names, name)
		}
	}

	return names, nil
}
