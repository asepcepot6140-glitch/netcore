package pppoe

type Pool struct {
	Name   string `json:"name"`
	Ranges string `json:"ranges"`
}

func ListPools(userID uint, mikrotikID uint64) ([]Pool, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(
		"/ip/pool/print",
		"=.proplist=name,ranges",
	)
	if err != nil {
		return nil, err
	}

	pools := make([]Pool, 0, len(reply.Re))

	for _, row := range reply.Re {
		pools = append(pools, Pool{
			Name:   row.Map["name"],
			Ranges: row.Map["ranges"],
		})
	}

	return pools, nil
}
