package web

const Index = `<!DOCTYPE html>
<html lang="en">
<html>
  <head>
    <meta charset="UTF-8">
    <script>{{ .ChartJs }}</script>
    <script>{{ .MainJs }}</script>
    <style>{{ .MainCss }}</style>
    <script>
      const jobsSuccessRatesArr = {{ .JobSuccessRatesArr }};
      const workflowsArr = {{ .WorkflowsArr }};
      window.onload = function() {
        const jobsLabels = jobsSuccessRatesArr.map(j => j.Key)
        const jobsDatasets = jobsSuccessRatesArr.map(j => j.Value)
        jobsSuccessRates('jobs-success-rates', jobsLabels, jobsDatasets);
        workflowsArr.forEach( workflow =>  {
          createCanvas(workflow.Id);
          const labels = workflow.Messages.map(m => m.Key);
          const datasets = workflow.Messages.map(m => m.Value);
          workflowMessages(workflow, labels, datasets);
        });
      };
    </script>
  </head>
  <body>

    <div id="timespan">{{ .Start }} ⇨ {{ .End }}</div>
    <div class="topWrapper">
      <div>
        <h3>Global Success Rate</h3>
        <div id="globalSuccessRate">
          {{ .GlobalSuccessRate }}%
        </div>
        <div>
            <h3>Success Rates per Workflow</h3>
            <div style="padding-left:80px">
              <table style="width:100%">
                {{ range .WorkflowSuccessRates }}
                <tr>
                  <td style="font-size:35px">{{ .Key }}:</td>
                  <td style="font-size:40px">{{ .Value }}%</td>
                </tr>
                {{ end }}
              </table>
            </div>
        </div>
      </div>
      <div class="light">
          <h3>Jobs Success Rates</h3>
          <canvas id="jobs-success-rates"></canvas>
      </div>
    </div>

    <div class="subSection light">
      <h3>Failed Tests per Workflow</h3>
      <div id="divWorkflowMessages" style="grid-template-columns:1fr 1fr 1fr;">
      </div>
    </div>
  </body>
</html>`
