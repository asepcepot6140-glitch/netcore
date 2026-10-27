package hotspot

import "sort"

// listNames mengumpulkan nama unik dari beberapa path RouterOS, terurut, untuk pilihan dropdown.
func listNames(userID uint, mikrotikID uint64, paths ...string) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	seen := make(map[string]struct{})
	for _, path := range paths {
		reply, err := client.Run(path+"/print", "=.proplist=name")
		if err != nil {
			return nil, err
		}
		for _, row := range reply.Re {
			if name := row.Map["name"]; name != "" {
				seen[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// ListPoolNames mengembalikan nama address pool dari /ip/pool.
func ListPoolNames(userID uint, mikrotikID uint64) ([]string, error) {
	return listNames(userID, mikrotikID, "/ip/pool")
}

// ListQueueNames mengembalikan nama queue simple dan queue tree (untuk parent queue).
func ListQueueNames(userID uint, mikrotikID uint64) ([]string, error) {
	return listNames(userID, mikrotikID, "/queue/simple", "/queue/tree")
}

// ListQueueTypeNames mengembalikan nama queue type dari /queue/type.
func ListQueueTypeNames(userID uint, mikrotikID uint64) ([]string, error) {
	return listNames(userID, mikrotikID, "/queue/type")
}
