# linkerd2-ci-metrics

This project generates a report for the CI builds at the [Linkerd2
repo](https://github.com/linkerd/linkerd2). 

Each day a cronjob (as defined in `.github/workflows/build-report.yml`) will
trigger the creation of the report, which is uploaded as an artifact to the
workflow, under the Actions tab.

The report is a zip file containing a single file `report.html` that is
self-contained, i.e. it doesn't depend on fetching external js libraries or any
other kind of asset.

### Reports

![top screenshot](https://github.com/linkerd/linkerd2-ci-metrics/blob/master/screenshots/top.png)
![bottom screenshot](https://github.com/linkerd/linkerd2-ci-metrics/blob/master/screenshots/bottom.png)

The first pane shows the global success of all the CI runs in the repo,
aggregating over all the workflow runs whose jobs weren't cancelled. Below
that rate is segregated per workflow.

The second plane shows the success rates per job, from less to more successful.

The bottom panes show the list of error messages captured through Github
annotations from more to less frequent. These are shown just for the workflows
that run integration tests: Kind integration, Cloud integration and Release.

### API Requests

The program makes use of Google's
[go-github](https://github.com/google/go-github) library, to hit the following
Github APIs:

```
# This gives us a list of check suite IDs:
GET /repos/linkerd/linkerd2/actions/runs

# For each check suite ID, this gives us the check run ID, job name, workflow
# ID, completion status and timestamps:
GET /repos/linkerd/linkerd2/check-suites/:check_suite_id/check-runs

# For each workflow ID, this gives us the workflow name
# (this can be cached across check runs):
GET https://api.github.com/repos/linkerd/linkerd2/actions/workflows/:workflow_id

# For each check run ID, we invoke the annotations API which gives us the file
# name and error message
GET repos/linkerd/linkerd2/check-runs/:check_run_id/annotations
```

### Authentication

The Github API requests are authenticated using the `REPORTS_TOKEN` secret, containing
a Personal Access Token belonging to l5d-bot.

### Testing

`go test /.cmd` will test that the html report is generated without errors, using
as inputs the list of jobs and annotations found under `cmd/testdata/jobs.json`
and `cmd/testdata/annotations.json`.

You can view the sample report generated with that data with `go test ./cmd -v`

## License

Copyright 2020, Linkerd Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
these files except in compliance with the License. You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
