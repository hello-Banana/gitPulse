package collector

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitCollector Git 数据采集器
type GitCollector struct {
	repoPath string
}

// CommitRecord 提交记录
type CommitRecord struct {
	Hash      string
	Author    string
	Email     string
	Time      time.Time
	Timezone  string
	Subject   string
	Additions int
	Deletions int
}

// CollectOptions 采集选项
type CollectOptions struct {
	Since         time.Time
	Until         time.Time
	Author        string
	ExcludeAuthor string
	Branch        string
	IgnoreMsg     string
	IncludeMerges bool
}

// NewGitCollector 创建 Git 采集器
func NewGitCollector(repoPath string) (*GitCollector, error) {
	return &GitCollector{
		repoPath: repoPath,
	}, nil
}

// CollectCommits 采集提交数据 - 使用 git 命令替代 go-git 提升性能
func (g *GitCollector) CollectCommits(opts *CollectOptions) ([]CommitRecord, error) {
	if opts == nil {
		opts = &CollectOptions{}
	}

	// 构建 git log 命令
	args := []string{"-C", g.repoPath, "log", "--no-merges"}

	// 使用更高效的输出格式
	args = append(args, "--format=%H%x00%an%x00%ae%x00%ai%x00%s")

	// 如果不需要变更统计，跳过统计计算
	if !opts.IncludeMerges {
		args = append(args, "--no-merges")
	}

	// 时间范围过滤
	if !opts.Since.IsZero() {
		args = append(args, "--since", opts.Since.Format("2006-01-02"))
	}
	if !opts.Until.IsZero() {
		args = append(args, "--until", opts.Until.Format("2006-01-02"))
	}

	// 分支
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	} else {
		args = append(args, "HEAD")
	}

	cmd := exec.Command("git", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建管道失败：%v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("执行 git 失败：%v", err)
	}

	var commits []CommitRecord
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\x00", 5)
		if len(parts) != 5 {
			continue
		}

		// 作者过滤
		if opts.Author != "" && parts[1] != opts.Author && parts[2] != opts.Author {
			continue
		}

		// 排除作者
		if opts.ExcludeAuthor != "" {
			if matchPattern(opts.ExcludeAuthor, parts[1]) || matchPattern(opts.ExcludeAuthor, parts[2]) {
				continue
			}
		}

		// 提交信息过滤
		if opts.IgnoreMsg != "" && strings.Contains(parts[4], opts.IgnoreMsg) {
			continue
		}

		// 解析时间 (格式：2024-01-15 10:30:00 +0800)
		t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[3])
		if err != nil {
			continue
		}

		commit := CommitRecord{
			Hash:    parts[0],
			Author:  parts[1],
			Email:   parts[2],
			Time:    t,
			Timezone: parts[3][len(parts[3])-5:],
			Subject: parts[4],
		}

		commits = append(commits, commit)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取输出失败：%v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git 命令失败：%v", err)
	}

	return commits, nil
}

// CollectCommitsWithStats 采集提交数据（包含变更统计）- 使用 git numstat
func (g *GitCollector) CollectCommitsWithStats(opts *CollectOptions) ([]CommitRecord, error) {
	commits, err := g.CollectCommits(opts)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return commits, nil
	}

	// 批量获取变更统计
	stats, err := g.getCommitStats(commits, opts)
	if err != nil {
		return nil, err
	}

	for i := range commits {
		if s, ok := stats[commits[i].Hash]; ok {
			commits[i].Additions = s[0]
			commits[i].Deletions = s[1]
		}
	}

	return commits, nil
}

// getCommitStats 批量获取提交的变更统计
func (g *GitCollector) getCommitStats(commits []CommitRecord, opts *CollectOptions) (map[string][2]int, error) {
	stats := make(map[string][2]int)

	args := []string{"-C", g.repoPath, "log", "--no-merges", "--numstat", "--format=COMMIT:%H"}

	if !opts.Since.IsZero() {
		args = append(args, "--since", opts.Since.Format("2006-01-02"))
	}
	if !opts.Until.IsZero() {
		args = append(args, "--until", opts.Until.Format("2006-01-02"))
	}

	args = append(args, "HEAD")

	cmd := exec.Command("git", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return stats, err
	}

	if err := cmd.Start(); err != nil {
		return stats, err
	}

	scanner := bufio.NewScanner(stdout)
	var currentHash string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "COMMIT:") {
			currentHash = strings.TrimPrefix(line, "COMMIT:")
			continue
		}

		if currentHash == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if parts[0] != "-" && parts[1] != "-" {
				adds, _ := strconv.Atoi(parts[0])
				dels, _ := strconv.Atoi(parts[1])
				s := stats[currentHash]
				s[0] += adds
				s[1] += dels
				stats[currentHash] = s
			}
		}
	}

	cmd.Wait()
	return stats, nil
}

// GetAuthors 获取所有作者列表
func (g *GitCollector) GetAuthors() ([]string, error) {
	cmd := exec.Command("git", "-C", g.repoPath, "log", "--format=%an <%ae>", "--all")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	authorMap := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			authorMap[line] = true
		}
	}

	var authors []string
	for author := range authorMap {
		authors = append(authors, author)
	}

	return authors, nil
}

// GetBranches 获取所有分支列表
func (g *GitCollector) GetBranches() ([]string, error) {
	cmd := exec.Command("git", "-C", g.repoPath, "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, b := range branches {
		if b != "" && !strings.Contains(b, " ") {
			result = append(result, b)
		}
	}

	return result, nil
}

// matchPattern 简单模式匹配（支持 * 通配符）
func matchPattern(pattern, str string) bool {
	if pattern == "" {
		return false
	}

	if pattern[0] != '*' && pattern[len(pattern)-1] != '*' {
		return pattern == str
	}

	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}

	if pattern[0] == '*' {
		suffix := pattern[1:]
		return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
	}

	return false
}

// GetFirstCommitDate 获取首次提交日期
func (g *GitCollector) GetFirstCommitDate() (time.Time, error) {
	cmd := exec.Command("git", "-C", g.repoPath, "log", "--reverse", "--format=%ai", "-n", "1")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(output)))
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// GetLatestCommitDate 获取最后提交日期
func (g *GitCollector) GetLatestCommitDate() (time.Time, error) {
	cmd := exec.Command("git", "-C", g.repoPath, "log", "-1", "--format=%ai")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(output)))
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// GetCurrentEmail 获取当前用户邮箱
func (g *GitCollector) GetCurrentEmail() (string, error) {
	cmd := exec.Command("git", "-C", g.repoPath, "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
