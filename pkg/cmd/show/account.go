package showcmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/output"
	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/runtime"
)

func newAccountCmd() *cobra.Command {
	var username string
	var windowDays int

	cmd := &cobra.Command{
		Use:   "account",
		Short: "Show account and usage information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowAccount(cmd, username, windowDays)
		},
	}

	cmd.Flags().StringVar(&username, "user", "", "User to summarize; defaults to the current connection user")
	cmd.Flags().IntVar(&windowDays, "window", 7, "Lookback window (days) for activity metrics")
	return cmd
}

func runShowAccount(cmd *cobra.Command, username string, windowDays int) error {
	if windowDays <= 0 {
		windowDays = 7
	}

	ctxInfo, err := runtime.RequireActiveContext(cmd.Context())
	if err != nil {
		return err
	}

	effectiveUser := strings.TrimSpace(username)
	if effectiveUser == "" {
		effectiveUser = ctxInfo.User
	}
	if effectiveUser == "" {
		return fmt.Errorf("user not provided and no user configured in the current context")
	}

	summary := &accountSummary{
		Context: contextSummary{
			Account:     ctxInfo.Account,
			AccountURL:  ctxInfo.AccountURL,
			User:        ctxInfo.User,
			Role:        ctxInfo.Role,
			Warehouse:   ctxInfo.Warehouse,
			Database:    ctxInfo.Database,
			Schema:      ctxInfo.Schema,
			Description: ctxInfo.Description,
		},
		WindowDays: windowDays,
	}

	ctx := cmd.Context()

	progress := newSpinner(cmd.ErrOrStderr())
	progress.Start("Fetching user info...")
	if err := summary.collectUserInfo(ctx, ctxInfo, effectiveUser); err != nil {
		return fmt.Errorf("fetch user info: %w", err)
	}
	progress.Start("Fetching login history...")
	if err := summary.collectLoginInfo(ctx, ctxInfo, effectiveUser); err != nil {
		return fmt.Errorf("fetch login info: %w", err)
	}
	progress.Start("Collecting query statistics...")
	if err := summary.collectQueryStats(ctx, ctxInfo, effectiveUser); err != nil {
		return fmt.Errorf("fetch query stats: %w", err)
	}
	progress.Start("Summarizing warehouse usage...")
	if err := summary.collectWarehouseUsage(ctx, ctxInfo, effectiveUser); err != nil {
		return fmt.Errorf("fetch warehouse usage: %w", err)
	}
	progress.Stop()
	printHumanSummary(cmd, summary)

	if structuredOutputRequested(cmd) {
		return output.Print(cmd, summary)
	}
	return nil
}

type accountSummary struct {
	Context       contextSummary   `json:"context"`
	WindowDays    int              `json:"windowDays"`
	User          *userInfo        `json:"user"`
	LoginActivity *loginStats      `json:"loginActivity"`
	QueryActivity *queryStats      `json:"queryActivity"`
	WarehouseTop  []warehouseUsage `json:"warehouseUsage"`
}

type contextSummary struct {
	Account     string `json:"account,omitempty"`
	AccountURL  string `json:"accountUrl,omitempty"`
	User        string `json:"user,omitempty"`
	Role        string `json:"role,omitempty"`
	Warehouse   string `json:"warehouse,omitempty"`
	Database    string `json:"database,omitempty"`
	Schema      string `json:"schema,omitempty"`
	Description string `json:"description,omitempty"`
}

type userInfo struct {
	Name             string `json:"name"`
	LoginName        string `json:"loginName"`
	DisplayName      string `json:"displayName,omitempty"`
	Email            string `json:"email,omitempty"`
	CreatedOn        string `json:"createdOn,omitempty"`
	LastSuccessLogin string `json:"lastSuccessLogin,omitempty"`
	Disabled         bool   `json:"disabled"`
}

type loginStats struct {
	LoginsLastWindow int    `json:"loginsInWindow"`
	LastLogin        string `json:"lastLogin,omitempty"`
}

type queryStats struct {
	Queries      int64   `json:"queries"`
	TotalSeconds float64 `json:"totalSeconds"`
	BytesScanned int64   `json:"bytesScanned"`
}

type warehouseUsage struct {
	Warehouse   string  `json:"warehouse"`
	CreditsUsed float64 `json:"creditsUsed,omitempty"`
	Queries     int64   `json:"queries,omitempty"`
}

func (s *accountSummary) collectUserInfo(ctx context.Context, info *config.Context, username string) error {
	stmt := fmt.Sprintf(`select name, login_name, display_name, email, created_on, last_success_login, disabled
from snowflake.account_usage.users
where name = %s
order by created_on desc
limit 1`, quoteLiteral(strings.ToUpper(username)))

	rows, err := runQueryFn(ctx, info, stmt)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		s.User = &userInfo{Name: strings.ToUpper(username)}
		return nil
	}
	row := rows[0]
	s.User = &userInfo{
		Name:             getString(row, "NAME"),
		LoginName:        getString(row, "LOGIN_NAME"),
		DisplayName:      getString(row, "DISPLAY_NAME"),
		Email:            getString(row, "EMAIL"),
		CreatedOn:        getString(row, "CREATED_ON"),
		LastSuccessLogin: getString(row, "LAST_SUCCESS_LOGIN"),
		Disabled:         strings.EqualFold(getString(row, "DISABLED"), "true"),
	}
	return nil
}

func (s *accountSummary) collectLoginInfo(ctx context.Context, info *config.Context, username string) error {
	stmt := fmt.Sprintf(`select count(*) as login_count, max(event_timestamp) as last_login
from snowflake.account_usage.login_history
where user_name = %s
  and event_timestamp >= dateadd(day, -%d, current_timestamp())`, quoteLiteral(strings.ToUpper(username)), s.WindowDays)

	rows, err := runQueryFn(ctx, info, stmt)
	if err != nil {
		return err
	}
	stat := &loginStats{}
	if len(rows) > 0 {
		stat.LoginsLastWindow = int(getInt64(rows[0], "LOGIN_COUNT"))
		stat.LastLogin = getString(rows[0], "LAST_LOGIN")
	}
	s.LoginActivity = stat
	return nil
}

func (s *accountSummary) collectQueryStats(ctx context.Context, info *config.Context, username string) error {
	stmt := fmt.Sprintf(`select
  count(*) as query_count,
  coalesce(sum(total_elapsed_time),0) as total_elapsed_time,
  coalesce(sum(bytes_scanned),0) as bytes_scanned
from snowflake.account_usage.query_history
where user_name = %s
  and start_time >= dateadd(day, -%d, current_timestamp())`, quoteLiteral(strings.ToUpper(username)), s.WindowDays)

	rows, err := runQueryFn(ctx, info, stmt)
	if err != nil {
		return err
	}
	s.QueryActivity = &queryStats{}
	if len(rows) > 0 {
		s.QueryActivity.Queries = getInt64(rows[0], "QUERY_COUNT")
		elapsedMs := getFloat64(rows[0], "TOTAL_ELAPSED_TIME")
		s.QueryActivity.TotalSeconds = elapsedMs / 1000
		s.QueryActivity.BytesScanned = getInt64(rows[0], "BYTES_SCANNED")
	}
	return nil
}

func (s *accountSummary) collectWarehouseUsage(ctx context.Context, info *config.Context, username string) error {
	stmt := fmt.Sprintf(`select warehouse_name,
	       coalesce(sum(credits_used),0) as credits
from snowflake.account_usage.query_history
where user_name = %s
  and warehouse_name is not null
  and start_time >= dateadd(day, -%d, current_timestamp())
group by 1
order by credits desc
limit 5`, quoteLiteral(strings.ToUpper(username)), s.WindowDays)

	rows, err := runQueryFn(ctx, info, stmt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid identifier 'credits_used'") {
			return s.collectWarehouseFallback(ctx, info, username)
		}
		return err
	}
	usages := make([]warehouseUsage, 0, len(rows))
	for _, row := range rows {
		usages = append(usages, warehouseUsage{
			Warehouse:   getString(row, "WAREHOUSE_NAME"),
			CreditsUsed: getFloat64(row, "CREDITS"),
		})
	}
	s.WarehouseTop = usages
	return nil
}

// collectWarehouseFallback runs when credits columns are unavailable.
func (s *accountSummary) collectWarehouseFallback(ctx context.Context, info *config.Context, username string) error {
	stmt := fmt.Sprintf(`select warehouse_name, count(*) as queries
from snowflake.account_usage.query_history
where user_name = %s
  and warehouse_name is not null
  and start_time >= dateadd(day, -%d, current_timestamp())
group by 1
order by queries desc
limit 5`, quoteLiteral(strings.ToUpper(username)), s.WindowDays)

	rows, err := runQueryFn(ctx, info, stmt)
	if err != nil {
		return err
	}
	usages := make([]warehouseUsage, 0, len(rows))
	for _, row := range rows {
		usages = append(usages, warehouseUsage{
			Warehouse: getString(row, "WAREHOUSE_NAME"),
			Queries:   getInt64(row, "QUERIES"),
		})
	}
	s.WarehouseTop = usages
	return nil
}

// progressPrinter writes a single overwriting status line to stderr.
type spinnerProgress struct {
	writer  io.Writer
	frames  []rune
	msg     string
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func newSpinner(w io.Writer) *spinnerProgress {
	return &spinnerProgress{
		writer: w,
		frames: []rune{'|', '/', '-', '\\'},
	}
}

func (s *spinnerProgress) Start(message string) {
	if s == nil || s.writer == nil {
		return
	}
	s.mu.Lock()
	s.msg = message
	s.mu.Unlock()
	if s.running {
		return
	}
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.running = true
	go s.loop()
}

func (s *spinnerProgress) loop() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	idx := 0
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			fmt.Fprintf(s.writer, "\r%c %s", s.frames[idx], msg)
			idx = (idx + 1) % len(s.frames)
		case <-s.stopCh:
			fmt.Fprint(s.writer, "\r")
			close(s.doneCh)
			return
		}
	}
}

func (s *spinnerProgress) Stop() {
	if s == nil || !s.running {
		return
	}
	close(s.stopCh)
	<-s.doneCh
	s.running = false
	fmt.Fprint(s.writer, "\r\n")
}

func quoteLiteral(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

func getString(row map[string]any, key string) string {
	for k, v := range row {
		if strings.EqualFold(k, key) {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func getInt64(row map[string]any, key string) int64 {
	for k, v := range row {
		if strings.EqualFold(k, key) {
			switch val := v.(type) {
			case int64:
				return val
			case int:
				return int64(val)
			case float64:
				return int64(val)
			case string:
				var parsed int64
				fmt.Sscan(val, &parsed)
				return parsed
			}
		}
	}
	return 0
}

func getFloat64(row map[string]any, key string) float64 {
	for k, v := range row {
		if strings.EqualFold(k, key) {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case string:
				var parsed float64
				fmt.Sscan(val, &parsed)
				return parsed
			}
		}
	}
	return 0
}

func printHumanSummary(cmd *cobra.Command, summary *accountSummary) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "Account summary")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-20s %s\n", "Account:", summary.Context.Account)
	fmt.Fprintf(w, "%-20s %s\n", "Account URL:", summary.Context.AccountURL)
	fmt.Fprintf(w, "%-20s %s\n", "User:", summary.Context.User)
	fmt.Fprintf(w, "%-20s %s\n", "Role:", summary.Context.Role)
	fmt.Fprintf(w, "%-20s Last %d days\n", "Window:", summary.WindowDays)
	fmt.Fprintln(w)

	if summary.User != nil {
		fmt.Fprintln(w, "User profile:")
		fmt.Fprintf(w, "    %-22s %s\n", "Login name:", summary.User.LoginName)
		fmt.Fprintf(w, "    %-22s %s\n", "Email:", summary.User.Email)
		fmt.Fprintf(w, "    %-22s %s\n", "Created:", humanTime(summary.User.CreatedOn))
		fmt.Fprintf(w, "    %-22s %s\n", "Last success login:", humanTime(summary.User.LastSuccessLogin))
		fmt.Fprintf(w, "    %-22s %t\n", "Disabled:", summary.User.Disabled)
		fmt.Fprintln(w)
	}

	if summary.LoginActivity != nil {
		fmt.Fprintln(w, "Login activity:")
		fmt.Fprintf(w, "    %-22s %s\n", "Logins:", humanCount(int64(summary.LoginActivity.LoginsLastWindow)))
		fmt.Fprintf(w, "    %-22s %s\n", "Last login:", humanTime(summary.LoginActivity.LastLogin))
		fmt.Fprintln(w)
	}

	if summary.QueryActivity != nil {
		fmt.Fprintln(w, "Query activity:")
		fmt.Fprintf(w, "    %-22s %s\n", "Queries:", humanCount(summary.QueryActivity.Queries))
		fmt.Fprintf(w, "    %-22s %s\n", "Runtime:", humanDuration(summary.QueryActivity.TotalSeconds))
		fmt.Fprintf(w, "    %-22s %s\n", "Bytes scanned:", humanBytes(summary.QueryActivity.BytesScanned))
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Warehouse usage:")
	if len(summary.WarehouseTop) == 0 {
		fmt.Fprintln(w, "    (no data)")
	} else {
		for _, wh := range summary.WarehouseTop {
			metric := "n/a"
			if wh.CreditsUsed > 0 {
				metric = fmt.Sprintf("%.2f credits", wh.CreditsUsed)
			} else if wh.Queries > 0 {
				metric = fmt.Sprintf("%s queries", humanCount(wh.Queries))
			}
			fmt.Fprintf(w, "    %s\t%s\n", wh.Warehouse, metric)
		}
	}

	w.Flush()
	fmt.Fprintln(cmd.OutOrStdout())
}

func structuredOutputRequested(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if flag := cmd.Flags().Lookup("output"); flag != nil && flag.Changed {
		return true
	}
	if flag := cmd.InheritedFlags().Lookup("output"); flag != nil && flag.Changed {
		return true
	}
	return false
}

func humanCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < len(s); i += 3 {
		b.WriteString(",")
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func humanDuration(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	d := time.Duration(seconds * float64(time.Second))
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	parts := make([]string, 0, 3)
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
}

func humanBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(b)
	var unit string
	for _, u := range units {
		value /= 1024
		unit = u
		if value < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

func humanTime(raw string) string {
	if raw == "" {
		return "n/a"
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts.Format("2006-01-02 15:04:05 MST")
	}
	return raw
}
