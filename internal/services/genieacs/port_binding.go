package genieacs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

// Newer ZTE firmware (e.g. F670L) keeps LAN binding in a device-wide table, one entry per WAN connection.
const portBindingPath = "InternetGatewayDevice.X_ZTE-COM_PortBinding"

// usesPortBinding reports connections without their own bind parameter; they rely on the PortBinding table.
func usesPortBinding(connection map[string]any) bool {
	_, huawei := connection["X_HW_LANBIND"]
	return !huawei && !hasLanBindKey(connection)
}

// portBindingEntry finds the table entry bound to the WAN connection at wanPath.
func portBindingEntry(table map[string]any, wanPath string) (index string, entry map[string]any) {
	for _, key := range numericKeys(table) {
		candidate, _ := table[key].(map[string]any)
		if getValue(candidate, "WANInterface") == wanPath {
			return key, candidate
		}
	}
	return "", nil
}

// checkPortBindingReady rejects a connection the device would refuse to bind (type not set yet).
func checkPortBindingReady(connType string) error {
	if connType == "" || strings.EqualFold(connType, "Unconfigured") {
		return fmt.Errorf("%w: set the connection type before binding LAN ports", ErrInvalidParameter)
	}
	return nil
}

// portBindingUpdates builds the writes that bind the selected ports to wanPath, creating the entry when missing.
func portBindingUpdates(ctx context.Context, base, deviceID, wanPath, raw string) ([][]any, error) {
	selected, err := parseLanBindLabels(raw)
	if err != nil {
		return nil, err
	}
	lan := strings.Join(lanBindPaths(selected), ",")

	table, err := fetchObject(ctx, base, deviceID, portBindingPath)
	if errors.Is(err, ErrTargetNotFound) {
		return nil, fmt.Errorf("%w: lan_bind", ErrParameterNotSupported)
	}
	if err != nil {
		return nil, err
	}

	if index, entry := portBindingEntry(table, wanPath); entry != nil {
		return [][]any{{portBindingPath + "." + index + ".LANInterface", lan, reportedType(entry, "LANInterface")}}, nil
	}
	if lan == "" {
		return nil, nil
	}

	known := make(map[string]bool)
	for _, key := range numericKeys(table) {
		known[key] = true
	}
	applied, err := postTask(ctx, base, deviceID, map[string]any{
		"name":       "addObject",
		"objectName": portBindingPath,
	})
	if err != nil {
		return nil, err
	}
	if !applied {
		return nil, errors.New("the device did not confirm the new port binding entry")
	}
	index, err := waitNewInstance(ctx, base, deviceID, portBindingPath, known)
	if err != nil {
		return nil, err
	}

	entryPath := portBindingPath + "." + index
	// A new entry must get its WANInterface first; on failure it is removed so no empty entry stays behind.
	if _, err := postTask(ctx, base, deviceID, map[string]any{
		"name":            "setParameterValues",
		"parameterValues": [][]any{{entryPath + ".WANInterface", wanPath, typeString}},
	}); err != nil {
		if _, delErr := postTask(ctx, base, deviceID, map[string]any{
			"name":       "deleteObject",
			"objectName": entryPath,
		}); delErr != nil {
			log.Printf("genieacs port binding cleanup error: %v", delErr)
		}
		return nil, err
	}
	return [][]any{{entryPath + ".LANInterface", lan, typeString}}, nil
}

// removePortBindings deletes table entries that point at a WAN connection that no longer exists.
// removed reports whether any entry was deleted.
func removePortBindings(ctx context.Context, base, deviceID, wanPath string) (removed bool, err error) {
	table, err := fetchObject(ctx, base, deviceID, portBindingPath)
	if errors.Is(err, ErrTargetNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, key := range numericKeys(table) {
		entry, _ := table[key].(map[string]any)
		if getValue(entry, "WANInterface") != wanPath {
			continue
		}
		if _, err := postTask(ctx, base, deviceID, map[string]any{
			"name":       "deleteObject",
			"objectName": portBindingPath + "." + key,
		}); err != nil {
			return removed, err
		}
		removed = true
	}
	return removed, nil
}

// reportedType returns the device-reported type of a child parameter, defaulting to string.
func reportedType(node map[string]any, name string) string {
	if object, ok := node[name].(map[string]any); ok {
		if reported, _ := object["_type"].(string); reported != "" {
			return reported
		}
	}
	return typeString
}
