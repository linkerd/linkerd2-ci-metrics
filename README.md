# linkerd2-ci-metrics

This project generates a report for the CI builds at the [Linkerd2
repo](https://github.com/linkerd/linkerd2). 

Whenever a build finishes in that repo, the `trigger-report` Github event is
raised, which is caught by the current project's `build-report.yml` workflow.
The report is uploaded as an artifact to the workflow, under the Actions tab.

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
