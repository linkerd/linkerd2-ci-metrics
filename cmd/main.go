package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alpeb/linkerd2-ci-metrics/cmd/pairlist"
	"github.com/alpeb/linkerd2-ci-metrics/cmd/web"
	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
)

const (
	owner      = "alpeb"
	repo       = "linkerd2"
	tokenLabel = "GITHUB_TOKEN"
)

var (
	completed      = "completed"
	all            = "all"
	ctx            context.Context
	client         *github.Client
	optBigListPage = &github.ListOptions{PerPage: 100}
	workflows      = map[int64]string{}
	now            = time.Now()
	monthAgo       = now.AddDate(0, -1, 0)
)

type JobRun struct {
	Workflow   string
	Job        string
	Conclusion string
	Started    github.Timestamp
	Completed  github.Timestamp
}

type ErrorAnn struct {
	JobRun
	Path      string
	StartLine int
	EndLine   int
	Message   string
}

type WorkflowWithMessages struct {
	Id       string
	Name     string
	Messages pairlist.PairList
}

type Page struct {
	ChartJs              template.JS
	MainJs               template.JS
	MainCss              template.CSS
	JobSuccessRatesArr   template.JS
	WorkflowsArr         template.JS
	Start                string
	End                  string
	GlobalSuccessRate    int
	WorkflowSuccessRates pairlist.PairList
}

// getWorkflowName returns the label for the workflow corresponding to workflowID.
// If that ID hasn't been cached yet, a call to the Github API is made
func getWorkflowName(workflowID int64) (string, error) {
	name, ok := workflows[workflowID]
	if ok {
		return name, nil
	}
	workflow, _, err := client.Actions.GetWorkflowByID(ctx, owner, repo, workflowID)
	if err != nil {
		return "", err
	}
	workflows[workflowID] = *workflow.Name
	return *workflow.Name, nil
}

// getAnnotations returns the list of annotations for the checkRunID
// Generic annotations with messages like "Process completed with exit code #"
// are not considered, neither are the jobs that were canceled because a sibling
// job didn't complete successfully.
func getAnnotations(checkRunID int64, job JobRun) ([]ErrorAnn, error) {
	annotations, _, err := client.Checks.ListCheckRunAnnotations(ctx, owner, repo, checkRunID, optBigListPage)
	if err != nil {
		return nil, err
	}

	var errorAnns []ErrorAnn
	for _, ann := range annotations {
		if strings.Contains(ann.GetMessage(), "Process completed with exit code") ||
			strings.Contains(ann.GetMessage(), "The job was canceled") {
			continue
		}
		errorAnn := ErrorAnn{
			JobRun:    job,
			Path:      ann.GetPath(),
			StartLine: ann.GetStartLine(),
			EndLine:   ann.GetEndLine(),
			Message:   ann.GetMessage(),
		}
		errorAnns = append(errorAnns, errorAnn)
	}
	return errorAnns, nil
}

// getJobRuns returns the list of jobs and annotations for the given checkSuiteID and
// workflowName. Only the workflows that have been completed, haven't been cancelled
// and started during the last month are returned.
func getJobRuns(checkSuiteID int64, workflowName string) ([]JobRun, []ErrorAnn, error) {
	opt := &github.ListCheckRunsOptions{Status: &completed, Filter: &all}
	checkRuns, _, err := client.Checks.ListCheckRunsCheckSuite(ctx, owner, repo, int64(checkSuiteID), opt)
	if err != nil {
		return nil, nil, err
	}
	allChecksCurrent := false
	var jobs []JobRun
	var allAnns []ErrorAnn
	for _, checkRun := range checkRuns.CheckRuns {
		if checkRun.GetConclusion() == "cancelled" {
			continue
		}

		job := JobRun{
			Workflow:   workflowName,
			Job:        checkRun.GetName(),
			Conclusion: checkRun.GetConclusion(),
			Started:    checkRun.GetStartedAt(),
			Completed:  checkRun.GetCompletedAt(),
		}
		allChecksCurrent = allChecksCurrent || job.Started.After(monthAgo)

		jobs = append(jobs, job)

		anns, err := getAnnotations(checkRun.GetID(), job)
		if err != nil {
			return nil, nil, err
		}
		allAnns = append(allAnns, anns...)
	}

	if !allChecksCurrent {
		return nil, nil, nil
	}

	return jobs, allAnns, nil
}

// getData builds the list of jobs and annotations for the current repo,
// calling the Github API
func getData() ([]JobRun, []ErrorAnn, error) {
	token, ok := os.LookupEnv(tokenLabel)
	if !ok {
		return nil, nil, fmt.Errorf("%s env var required", tokenLabel)
	}

	ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client = github.NewClient(tc)
	opt := optBigListPage
	var jobs []JobRun
	var annotations []ErrorAnn
out:
	for {
		runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opt)
		if err != nil {
			return nil, nil, err
		}

		var workflowJobs []JobRun
		var workflowAnnotations []ErrorAnn
		for _, run := range runs.WorkflowRuns {
			if run.GetConclusion() == "cancelled" {
				continue
			}
			url := run.GetCheckSuiteURL()
			checkSuiteIDstr := url[strings.LastIndex(url, "/")+1:]
			checkSuiteID, err := strconv.Atoi(checkSuiteIDstr)
			if err != nil {
				continue
			}

			workflowURL := run.GetWorkflowURL()
			workflowIDstr := workflowURL[strings.LastIndex(workflowURL, "/")+1:]
			workflowID, err := strconv.Atoi(workflowIDstr)
			if err != nil {
				continue
			}
			workflowName, err := getWorkflowName(int64(workflowID))
			if err != nil {
				return nil, nil, err
			}

			jobRuns, jobAnnotations, err := getJobRuns(int64(checkSuiteID), workflowName)
			if err != nil {
				return nil, nil, err
			}
			if jobRuns == nil {
				break out
			}
			workflowJobs = append(workflowJobs, jobRuns...)
			workflowAnnotations = append(workflowAnnotations, jobAnnotations...)
		}

		jobs = append(jobs, workflowJobs...)
		annotations = append(annotations, workflowAnnotations...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return jobs, annotations, nil
}

func getWorkflowMessages(workflow string, annotations []ErrorAnn) pairlist.PairList {
	messages := map[string]int{}
	for _, ann := range annotations {
		if ann.Workflow != workflow {
			continue
		}
		count := messages[ann.Message]
		count++
		messages[ann.Message] = count
	}

	return pairlist.RankByValue(messages, true)
}

// getJobSuccessRates returns a json array for the job success rates
// ordered from least to most sucessful, where the jobs with a 100%
// success rates are grouped into "Others" at the end
func getJobSuccessRates(runs []JobRun) ([]byte, error) {
	totalRuns := make(map[string]int)
	successes := make(map[string]int)
	for _, run := range runs {
		count := totalRuns[run.Job]
		count++
		totalRuns[run.Job] = count
		if run.Conclusion == "success" {
			count = successes[run.Job]
			count++
			successes[run.Job] = count
		}
	}
	for job, num := range successes {
		if successes[job] == totalRuns[job] && job != "Others" {
			delete(successes, job)
			successes["Others"] = 100
			totalRuns["Others"] = 100
			continue
		}
		successes[job] = num * 100 / totalRuns[job]
	}
	json, err := json.Marshal(pairlist.RankByValue(successes, false))
	if err != nil {
		return nil, err
	}
	return json, nil
}

// getWorkflowSuccessRates returns the global success rate and the success rate
// for each workflow (ordered from less to more successful)
func getWorkflowSuccessRates(runs []JobRun) (int, pairlist.PairList) {
	totalRuns := 0
	totalSuccesses := 0
	totalRunsPerJob := make(map[string]int)
	successes := make(map[string]int)
	for _, run := range runs {
		totalRuns++
		count := totalRunsPerJob[run.Workflow]
		count++
		totalRunsPerJob[run.Workflow] = count
		if run.Conclusion == "success" {
			totalSuccesses++
			count = successes[run.Workflow]
			count++
			successes[run.Workflow] = count
		}
	}
	for workflow, num := range successes {
		successes[workflow] = num * 100 / totalRunsPerJob[workflow]
	}
	return totalSuccesses * 100 / totalRuns, pairlist.RankByValue(successes, false)
}

// processData retrieves all the CI success and error message metrics and
// displays them in an index.html file
func processData(jobs []JobRun, annotations []ErrorAnn) error {
	jobSuccessRatesJson, err := getJobSuccessRates(jobs)
	if err != nil {
		return err
	}

	setWorkflows := make(map[string]struct{})
	for _, ann := range annotations {
		setWorkflows[ann.Workflow] = struct{}{}
	}

	messages := []WorkflowWithMessages{}
	for workflow, _ := range setWorkflows {
		m := WorkflowWithMessages{
			Id:       strings.ReplaceAll(workflow, " ", "-"),
			Name:     workflow,
			Messages: getWorkflowMessages(workflow, annotations),
		}
		messages = append(messages, m)
	}
	workflowsJson, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	globalSuccessRate, workflowSuccessRates := getWorkflowSuccessRates(jobs)

	tpl, err := template.New("index").Parse(web.Index)
	if err != nil {
		return err
	}
	data := Page{
		ChartJs:              template.JS(web.ChartJs),
		MainJs:               template.JS(web.MainJs),
		MainCss:              template.CSS(web.MainCss),
		JobSuccessRatesArr:   template.JS(jobSuccessRatesJson),
		WorkflowsArr:         template.JS(workflowsJson),
		Start:                monthAgo.Format(time.RFC822),
		End:                  now.Format(time.RFC822),
		GlobalSuccessRate:    globalSuccessRate,
		WorkflowSuccessRates: workflowSuccessRates,
	}
	if err := tpl.Execute(os.Stdout, data); err != nil {
		return err
	}
	return nil
}

func main() {
	jobs, annotations, err := getData()
	if err != nil {
		log.Fatal(err)
	}
	if err = processData(jobs, annotations); err != nil {
		log.Fatal(err)
	}
}
