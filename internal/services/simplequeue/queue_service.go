package simplequeue

import (
	"cmp"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-routeros/routeros/v3"
)

const (
	queuePath     = "/queue/simple"
	queueTypePath = "/queue/type"
	manglePath    = "/ip/firewall/mangle"

	queueProperties = ".id,name,target,max-limit,burst-limit,burst-threshold,burst-time,comment,packet-marks,limit-at,priority,bucket-size,queue,parent,disabled,dynamic,invalid,rate,bytes"

	maxValueLength = 1000
)

type Queue struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Target         string `json:"target"`
	MaxLimit       string `json:"max_limit"`
	BurstLimit     string `json:"burst_limit"`
	BurstThreshold string `json:"burst_threshold"`
	BurstTime      string `json:"burst_time"`
	Comment        string `json:"comment"`
	PacketMarks    string `json:"packet_marks"`
	LimitAt        string `json:"limit_at"`
	Priority       string `json:"priority"`
	BucketSize     string `json:"bucket_size"`
	QueueType      string `json:"queue_type"`
	Parent         string `json:"parent"`
	Disabled       bool   `json:"disabled"`
	Dynamic        bool   `json:"dynamic"`
	Invalid        bool   `json:"invalid"`
	// Rate dan Bytes berformat "upload/download" (bps dan byte).
	Rate  string `json:"rate"`
	Bytes string `json:"bytes"`
}

// Input: saat update, nil = tidak diubah dan "" = kembali ke bawaan RouterOS; saat create, nilai kosong dilewati.
type Input struct {
	Name           *string `json:"name"`
	Target         *string `json:"target"`
	MaxLimit       *string `json:"max_limit"`
	BurstLimit     *string `json:"burst_limit"`
	BurstThreshold *string `json:"burst_threshold"`
	BurstTime      *string `json:"burst_time"`
	Comment        *string `json:"comment"`
	PacketMarks    *string `json:"packet_marks"`
	LimitAt        *string `json:"limit_at"`
	Priority       *string `json:"priority"`
	BucketSize     *string `json:"bucket_size"`
	QueueType      *string `json:"queue_type"`
	Parent         *string `json:"parent"`
	Disabled       *bool   `json:"disabled"`
}

const rateAmount = `\d+(?:\.\d+)?[kKmMgG]?`

var (
	ratePattern     = regexp.MustCompile(`^` + rateAmount + `(?:/` + rateAmount + `)?$`)
	timePattern     = regexp.MustCompile(`^[0-9wdhms:.]+(?:/[0-9wdhms:.]+)?$`)
	bucketPattern   = regexp.MustCompile(`^\d+(?:\.\d+)?(?:/\d+(?:\.\d+)?)?$`)
	priorityPattern = regexp.MustCompile(`^[1-8](?:/[1-8])?$`)
)

// field memetakan satu properti opsional ke key RouterOS beserta aturannya.
type field struct {
	key   string
	label string
	value *string
	// reset dikirim saat nilai dikosongkan di edit agar router kembali ke bawaan.
	reset string
	// keepOnEmpty: nilai kosong saat edit diabaikan (properti tidak boleh dikosongkan).
	keepOnEmpty bool
	pattern     *regexp.Regexp
	example     string
}

func (in *Input) fields() []field {
	return []field{
		{key: "max-limit", label: "Max limit", value: in.MaxLimit, reset: "0", pattern: ratePattern, example: "5M/10M"},
		{key: "burst-limit", label: "Burst limit", value: in.BurstLimit, reset: "0", pattern: ratePattern, example: "10M/20M"},
		{key: "burst-threshold", label: "Burst threshold", value: in.BurstThreshold, reset: "0", pattern: ratePattern, example: "3M/6M"},
		{key: "burst-time", label: "Burst time", value: in.BurstTime, reset: "0s", pattern: timePattern, example: "20s/20s"},
		{key: "limit-at", label: "Limit at", value: in.LimitAt, reset: "0", pattern: ratePattern, example: "1M/2M"},
		{key: "priority", label: "Priority", value: in.Priority, reset: "8", pattern: priorityPattern, example: "8/8"},
		{key: "bucket-size", label: "Bucket size", value: in.BucketSize, reset: "0.1", pattern: bucketPattern, example: "0.1/0.1"},
		{key: "parent", label: "Parent", value: in.Parent, reset: "none"},
		{key: "packet-marks", label: "Packet marks", value: in.PacketMarks},
		{key: "comment", label: "Comment", value: in.Comment},
		{key: "queue", label: "Queue type", value: in.QueueType, keepOnEmpty: true},
	}
}

// secondsSuffix menambah "s" pada bagian waktu berupa angka saja ("0/0" -> "0s/0s").
func secondsSuffix(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		if part != "" && strings.Trim(part, "0123456789") == "" {
			parts[i] = part + "s"
		}
	}
	return strings.Join(parts, "/")
}

// normalizeTarget merapikan daftar target (dipisah koma) dan membuang entri kosong.
func normalizeTarget(value string) string {
	parts := strings.Split(value, ",")
	targets := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			targets = append(targets, part)
		}
	}
	return strings.Join(targets, ",")
}

// Normalize merapikan dan memvalidasi input; field nil dibiarkan.
func (in *Input) Normalize() error {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		in.Name = &name
	}
	if in.Target != nil {
		target := normalizeTarget(*in.Target)
		in.Target = &target
	}

	for _, f := range in.fields() {
		if f.value == nil {
			continue
		}

		value := strings.TrimSpace(*f.value)
		if f.pattern != nil {
			value = strings.Join(strings.Fields(value), "")
		}
		// Satu nama berarti dipakai untuk upload dan download (format RouterOS "up/down").
		if f.key == "queue" && value != "" && !strings.Contains(value, "/") {
			value += "/" + value
		}
		if value != "" && f.pattern != nil && !f.pattern.MatchString(value) {
			return &InputError{f.label + " tidak valid. Gunakan contoh: " + f.example}
		}
		if f.key == "burst-time" {
			value = secondsSuffix(value)
		}
		*f.value = value
	}

	for _, value := range []*string{in.Name, in.Target} {
		if value != nil && utf8.RuneCountInString(*value) > maxValueLength {
			return &InputError{"Nilai terlalu panjang"}
		}
	}
	for _, f := range in.fields() {
		if f.value != nil && utf8.RuneCountInString(*f.value) > maxValueLength {
			return &InputError{f.label + " terlalu panjang"}
		}
	}
	return nil
}

// BuildCreateArgs menyusun argumen /queue/simple/add; hanya field terisi yang dikirim.
func BuildCreateArgs(input Input) ([]string, error) {
	if err := input.Normalize(); err != nil {
		return nil, err
	}
	if input.Name == nil || *input.Name == "" {
		return nil, &InputError{"Name wajib diisi"}
	}
	if input.Target == nil || *input.Target == "" {
		return nil, &InputError{"Target wajib diisi"}
	}

	args := []string{queuePath + "/add", "=name=" + *input.Name, "=target=" + *input.Target}
	for _, f := range input.fields() {
		if f.value != nil && *f.value != "" {
			args = append(args, "="+f.key+"="+*f.value)
		}
	}
	if input.Disabled != nil && *input.Disabled {
		args = append(args, "=disabled=yes")
	}
	return args, nil
}

// BuildUpdateArgs menyusun argumen /queue/simple/set; field kosong dikembalikan ke bawaan RouterOS.
func BuildUpdateArgs(id string, input Input) ([]string, error) {
	if err := input.Normalize(); err != nil {
		return nil, err
	}
	if input.Name != nil && *input.Name == "" {
		return nil, &InputError{"Name tidak boleh kosong"}
	}
	if input.Target != nil && *input.Target == "" {
		return nil, &InputError{"Target tidak boleh kosong"}
	}

	args := []string{queuePath + "/set", "=.id=" + id}
	if input.Name != nil {
		args = append(args, "=name="+*input.Name)
	}
	if input.Target != nil {
		args = append(args, "=target="+*input.Target)
	}
	for _, f := range input.fields() {
		if f.value == nil {
			continue
		}
		value := *f.value
		if value == "" {
			if f.keepOnEmpty {
				continue
			}
			value = f.reset
		}
		args = append(args, "="+f.key+"="+value)
	}
	if input.Disabled != nil {
		args = append(args, "=disabled="+strconv.FormatBool(*input.Disabled))
	}

	if len(args) == 2 {
		return nil, ErrInvalidInput
	}
	return args, nil
}

// anyTarget adalah target "semua alamat"; RouterOS mengirimnya sebagai string kosong.
const anyTarget = "0.0.0.0/0"

func FromRow(row map[string]string) Queue {
	target := row["target"]
	if target == "" {
		target = anyTarget
	}

	return Queue{
		ID:             row[".id"],
		Name:           row["name"],
		Target:         target,
		MaxLimit:       row["max-limit"],
		BurstLimit:     row["burst-limit"],
		BurstThreshold: row["burst-threshold"],
		BurstTime:      row["burst-time"],
		Comment:        row["comment"],
		PacketMarks:    row["packet-marks"],
		LimitAt:        row["limit-at"],
		Priority:       row["priority"],
		BucketSize:     row["bucket-size"],
		QueueType:      row["queue"],
		Parent:         row["parent"],
		Disabled:       row["disabled"] == "true",
		Dynamic:        row["dynamic"] == "true",
		Invalid:        row["invalid"] == "true",
		Rate:           row["rate"],
		Bytes:          row["bytes"],
	}
}

// OrderNewestFirst membalik urutan RouterOS (terlama dulu) agar yang terbaru di atas, lalu menaruh
// queue dinamis (dibuat otomatis oleh PPPoE/hotspot, bisa ratusan) di bawah queue buatan user.
func OrderNewestFirst(items []Queue) {
	slices.Reverse(items)
	slices.SortStableFunc(items, func(a, b Queue) int {
		return cmp.Compare(boolRank(a.Dynamic), boolRank(b.Dynamic))
	})
}

func boolRank(value bool) int {
	if value {
		return 1
	}
	return 0
}

func List(userID uint, mikrotikID uint64) ([]Queue, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return fetchQueues(client)
}

// fetchQueues membaca semua queue lewat client yang sudah terhubung.
func fetchQueues(client *routeros.Client) ([]Queue, error) {
	reply, err := client.Run(queuePath+"/print", "=.proplist="+queueProperties)
	if err != nil {
		return nil, err
	}

	queues := make([]Queue, 0, len(reply.Re))
	for _, row := range reply.Re {
		queues = append(queues, FromRow(row.Map))
	}
	OrderNewestFirst(queues)
	return queues, nil
}

// ListTypeNames mengembalikan nama queue type dari /queue/type untuk dropdown.
func ListTypeNames(userID uint, mikrotikID uint64) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(queueTypePath+"/print", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(reply.Re))
	for _, row := range reply.Re {
		if name := row.Map["name"]; name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// ListPacketMarks mengembalikan nama packet-mark unik dari rule /ip/firewall/mangle.
// "no-mark" tidak disertakan karena itu bukan mark buatan user.
func ListPacketMarks(userID uint, mikrotikID uint64) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(manglePath+"/print", "=.proplist=new-packet-mark")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, row := range reply.Re {
		if mark := row.Map["new-packet-mark"]; mark != "" && mark != "no-mark" {
			seen[mark] = struct{}{}
		}
	}

	marks := make([]string, 0, len(seen))
	for mark := range seen {
		marks = append(marks, mark)
	}
	sort.Strings(marks)
	return marks, nil
}

// ListParentNames mengembalikan kandidat parent: queue simple buatan user (non-dinamis) dan queue tree.
func ListParentNames(userID uint, mikrotikID uint64) ([]string, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	seen := make(map[string]struct{})
	for _, query := range [][]string{
		{queuePath + "/print", "?dynamic=false", "=.proplist=name"},
		{"/queue/tree/print", "=.proplist=name"},
	} {
		reply, err := client.Run(query...)
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

// ensureEditable memastikan queue ada dan bukan dinamis; queue dinamis dikelola RouterOS.
func ensureEditable(client *routeros.Client, id string) error {
	reply, err := client.Run(queuePath+"/print", "?.id="+id, "=.proplist=.id,dynamic")
	if err != nil {
		return err
	}
	if len(reply.Re) == 0 {
		return ErrNotFound
	}
	if reply.Re[0].Map["dynamic"] == "true" {
		return ErrDynamic
	}
	return nil
}

func Create(userID uint, mikrotikID uint64, input Input) error {
	args, err := BuildCreateArgs(input)
	if err != nil {
		return err
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Run(args...)
	return mapRouterError(err)
}

func Update(userID uint, mikrotikID uint64, id string, input Input) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}
	args, err := BuildUpdateArgs(id, input)
	if err != nil {
		return err
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := ensureEditable(client, id); err != nil {
		return mapRouterError(err)
	}

	_, err = client.Run(args...)
	return mapRouterError(err)
}

func Delete(userID uint, mikrotikID uint64, id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := ensureEditable(client, id); err != nil {
		return mapRouterError(err)
	}

	_, err = client.Run(queuePath+"/remove", "=.id="+id)
	return mapRouterError(err)
}
