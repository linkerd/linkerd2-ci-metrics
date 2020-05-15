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

	"github.com/google/go-github/v31/github"
	"github.com/linkerd/linkerd2-ci-metrics/cmd/pairlist"
	"github.com/linkerd/linkerd2-ci-metrics/cmd/web"
	"golang.org/x/oauth2"
)

const (
	owner      = "linkerd"
	repo       = "linkerd2"
	tokenLabel = "GITHUB_TOKEN"

	// throttling requests to the Github API for retrieving annotations
	// at 1 req/sec, which puts us below the 5000 requests/hour limit
	rateLimit = time.Second
)

var (
	completed            = "completed"
	all                  = "all"
	interestingWorkflows = []string{"KinD integration", "Cloud integration", "Release"}
	ctx                  context.Context
	client               *github.Client
	optBigListPage       = &github.ListOptions{PerPage: 100}
	workflows            = map[int64]string{}
	now                  = time.Now()
	monthAgo             = now.AddDate(0, -1, 0)
	throttle             = time.Tick(rateLimit)
)

// JobRun holds the result state for a CI job, including the name of its
// parent workflow
type JobRun struct {
	Workflow   string
	Job        string
	Conclusion string
	Started    github.Timestamp
	Completed  github.Timestamp
}

// ErrorAnn holds the details of a CI run failure extracted from a
// Github annotation, and also points to its correspoinding JobRun
type ErrorAnn struct {
	JobRun
	Path      string
	StartLine int
	EndLine   int
	Message   string
}

// WorkflowWithMessages hold the details of a particular Workflow run,
// with its ID, Name, and list of error messages associated to it
type WorkflowWithMessages struct {
	Id       string
	Name     string
	Messages pairlist.PairList
}

// Page holds the data passed to the HTML template
type Page struct {
	ChartJS              template.JS
	MainJS               template.JS
	BootstrapCSS         template.CSS
	MainCSS              template.CSS
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

// isInteresting checks if the workflow is in the list of workflows
// (interestingWorkflows) for which we want to retrieve annotations
func isInteresting(workflow string) bool {
	for _, w := range interestingWorkflows {
		if workflow == w {
			return true
		}
	}
	return false
}

// getJobRuns returns the list of jobs and annotations for the given checkSuiteID and
// workflowName. Only the workflows that have been completed, haven't been cancelled
// and started during the last month are returned. The third argument returns true if
// there are more result pages available.
func getJobRuns(checkSuiteID int64, workflowName string) ([]JobRun, []ErrorAnn, bool, error) {
	opt := &github.ListCheckRunsOptions{Status: &completed, Filter: &all}
	checkRuns, _, err := client.Checks.ListCheckRunsCheckSuite(ctx, owner, repo, int64(checkSuiteID), opt)
	if err != nil {
		return nil, nil, false, err
	}
	// nextPage will be true if at least one job started this month.
	// Invalid workflows will have no jobs ran; for them nextPage is
	// true so that we still fetch the following page
	nextPage := len(checkRuns.CheckRuns) == 0
	var jobs []JobRun
	var allAnns []ErrorAnn
	for _, checkRun := range checkRuns.CheckRuns {
		nextPage = nextPage || checkRun.GetStartedAt().After(monthAgo)
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

		jobs = append(jobs, job)

		if !isInteresting(workflowName) {
			continue
		}

		<-throttle
		anns, err := getAnnotations(checkRun.GetID(), job)
		if err != nil {
			return nil, nil, false, err
		}
		allAnns = append(allAnns, anns...)
	}

	if !nextPage {
		return nil, nil, false, nil
	}

	return jobs, allAnns, true, nil
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

			jobRuns, jobAnnotations, nextPage, err := getJobRuns(int64(checkSuiteID), workflowName)
			if err != nil {
				return nil, nil, err
			}
			if !nextPage {
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
		messages[ann.Message]++
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
		totalRuns[run.Job]++
		if run.Conclusion == "success" {
			successes[run.Job]++
		}
	}
	for job, num := range successes {
		if totalRuns[job] == num && job != "Others" {
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
		totalRunsPerJob[run.Workflow]++
		if run.Conclusion == "success" {
			totalSuccesses++
			successes[run.Workflow]++
		}
	}
	for workflow, num := range successes {
		successes[workflow] = num * 100 / totalRunsPerJob[workflow]
	}

	if totalRuns > 0 {
		return totalSuccesses * 100 / totalRuns, pairlist.RankByValue(successes, false)
	}
	return 0, pairlist.PairList{}
}

// processData retrieves all the CI success and error message metrics and
// displays them in an index.html file
func processData(jobs []JobRun, annotations []ErrorAnn) error {
	jobSuccessRatesJSON, err := getJobSuccessRates(jobs)
	if err != nil {
		return err
	}

	setWorkflows := make(map[string]struct{})
	for _, ann := range annotations {
		setWorkflows[ann.Workflow] = struct{}{}
	}

	messages := []WorkflowWithMessages{}
	for workflow := range setWorkflows {
		m := WorkflowWithMessages{
			Id:       strings.ReplaceAll(workflow, " ", "-"),
			Name:     workflow,
			Messages: getWorkflowMessages(workflow, annotations),
		}
		messages = append(messages, m)
	}
	workflowsJSON, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	globalSuccessRate, workflowSuccessRates := getWorkflowSuccessRates(jobs)

	tpl, err := template.New("index").Parse(web.Index)
	if err != nil {
		return err
	}
	data := Page{
		ChartJS:              template.JS(web.ChartJS),
		MainJS:               template.JS(web.MainJS),
		BootstrapCSS:         template.CSS(web.BootstrapCSS),
		MainCSS:              template.CSS(web.MainCSS),
		JobSuccessRatesArr:   template.JS(jobSuccessRatesJSON),
		WorkflowsArr:         template.JS(workflowsJSON),
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
