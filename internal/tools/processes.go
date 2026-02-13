package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// processInfo holds information about a running process.
type processInfo struct {
	PID        int
	PPID       int
	Name       string
	User       string
	CPUPercent float64
	MemMB      float64
	State      string
}

// listProcessesFn is the function used to fetch processes. Swappable for testing.
var listProcessesFn = listProcesses

// listProcesses fetches the current process list using ps.
func listProcesses() ([]processInfo, error) {
	// ps -eo pid,ppid,user,state,%cpu,rss,comm
	// rss is in KB, we convert to MB
	cmd := exec.Command("ps", "-eo", "pid,ppid,user,state,%cpu,rss,comm")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps command failed: %w", err)
	}

	procs := make([]processInfo, 0, bytes.Count(out, []byte{'\n'}))
	scanner := bufio.NewScanner(bytes.NewReader(out))

	// Skip header line
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		p, ok := parsePSLine(line)
		if ok {
			procs = append(procs, p)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ps output: %w", err)
	}

	return procs, nil
}

// parsePSLine parses a single line of ps output.
// Format: PID PPID USER STATE %CPU RSS COMMAND
// The command field may contain spaces, so we parse the first 6 fields and take the rest.
func parsePSLine(line string) (processInfo, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return processInfo{}, false
	}

	fields := strings.Fields(line)
	if len(fields) < 7 {
		return processInfo{}, false
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return processInfo{}, false
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return processInfo{}, false
	}

	user := fields[2]
	state := fields[3]

	cpu, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		cpu = 0
	}

	rssKB, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		rssKB = 0
	}

	// Command is everything from field 6 onward (may contain spaces)
	name := strings.Join(fields[6:], " ")

	return processInfo{
		PID:        pid,
		PPID:       ppid,
		Name:       name,
		User:       user,
		CPUPercent: cpu,
		MemMB:      rssKB / 1024.0,
		State:      state,
	}, true
}

// HandleProcesses handles the processes tool call.
func HandleProcesses(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nameFilter := request.GetString("name", "")
	pidFilter := request.GetInt("pid", 0)

	procs, err := listProcessesFn()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list processes: %v", err)), nil
	}

	// Apply filters
	filtered := filterProcesses(procs, nameFilter, pidFilter)

	if len(filtered) == 0 {
		msg := "No processes found"
		if nameFilter != "" {
			msg += fmt.Sprintf(" matching name %q", nameFilter)
		}
		if pidFilter > 0 {
			msg += fmt.Sprintf(" matching PID %d", pidFilter)
		}
		return mcp.NewToolResultText(msg), nil
	}

	return mcp.NewToolResultText(formatProcessTable(filtered)), nil
}

// filterProcesses applies name and PID filters.
// When filtering by PID, includes the matched process and its children.
// When filtering by name, matches case-insensitively.
func filterProcesses(procs []processInfo, nameFilter string, pidFilter int) []processInfo {
	if nameFilter == "" && pidFilter == 0 {
		return procs
	}

	var matched []processInfo
	matchedPIDs := make(map[int]bool)

	if pidFilter > 0 {
		// Find the target process and collect its PID
		for _, p := range procs {
			if p.PID == pidFilter {
				matched = append(matched, p)
				matchedPIDs[p.PID] = true
				break
			}
		}
		// Find descendants (processes whose PPID is in matchedPIDs)
		// Single pass: picks up children, grandchildren, etc. when ordered by PID
		for _, p := range procs {
			if matchedPIDs[p.PPID] && !matchedPIDs[p.PID] {
				matched = append(matched, p)
				matchedPIDs[p.PID] = true
			}
		}
	} else {
		// Name filter (case-insensitive substring match)
		lowerFilter := strings.ToLower(nameFilter)
		for _, p := range procs {
			if strings.Contains(strings.ToLower(p.Name), lowerFilter) {
				matched = append(matched, p)
			}
		}
	}

	return matched
}

// formatProcessTable formats processes as an aligned text table.
func formatProcessTable(procs []processInfo) string {
	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "%-7s %-7s %-5s %-8s %-12s %-6s %s\n",
		"PID", "PPID", "CPU%", "MEM(MB)", "USER", "STATE", "NAME")
	sb.WriteString(strings.Repeat("-", 70))
	sb.WriteByte('\n')

	// Rows
	for _, p := range procs {
		fmt.Fprintf(&sb, "%-7d %-7d %-5.1f %-8.1f %-12s %-6s %s\n",
			p.PID, p.PPID, p.CPUPercent, p.MemMB, p.User, p.State, p.Name)
	}

	// Summary
	fmt.Fprintf(&sb, "\n%d processes", len(procs))

	return sb.String()
}
