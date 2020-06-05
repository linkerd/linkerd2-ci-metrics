package web

const Index = `<html lang="en">
<html>
  <head>
    <meta charset="UTF-8">
    <script>{{ .ChartJS }}</script>
    <script>{{ .MainJS }}</script>
    <style>{{ .BootstrapCSS }}</style>
    <style>{{ .MainCSS }}</style>
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
          chart = workflowMessages(workflow, labels, datasets);
	  chart.canvas.parentNode.style.height = 80 + workflow.Messages.length*70;
        });
      };
    </script>
  </head>
  <body>
    <nav id="topNav" class="navbar navbar-dark bg-primary">
      <div>Linkerd2 Integration Tests</div>
      <div id="timespan">{{ .Start }} ⇨ {{ .End }}</div>
    </nav>
    <div class="topWrapper">
      <div>
        <h3>Global Success Rate</h3>
        <div id="globalSuccessRate">
          {{ .GlobalSuccessRate }}%
        </div>
        <div id="ratesPerWorkflow" class="card shadow-lg p-3 mb-5 bg-white rounded">
          <div class="card-body">
            <h3>Success Rates per Workflow</h3>
            <div>
              <table>
                {{ range .WorkflowSuccessRates }}
                <tr>
                  <td class="left">{{ .Key }}:</td>
                  <td class="right">{{ .Value }}%</td>
                </tr>
                {{ end }}
              </table>
            </div>
          </div>
        </div>
      </div>
      <div class="card shadow-lg p-3 mb-5 bg-white rounded">
        <div class="card-body">
          <h3>Jobs Success Rates</h3>
            <canvas id="jobs-success-rates"></canvas>
		</div>
      </div>
    </div>

    <div class="subSection shadow-lg p-3 mb-5 bg-white rounded">
      <h3>Failed Tests per Workflow</h3>
      <div id="divWorkflowMessages">
      </div>
    </div>
  </body>
</html>`

/* vim: set tabstop=4:softtabstop=4:shiftwidth=4:expandtab */
