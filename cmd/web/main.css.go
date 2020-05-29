package web

const MainCSS = `
body {
  padding: 20px;
  padding-top: 0;
  font-family: arial;
}

h3 {
  font-size: 40px;
  font-weight: bold;
  text-align: center;
}

#topNav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  font-size: 30px;
  color: white;
}

#timespan {
  text-align:right;
}

.topWrapper {
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding-top: 20px;
}

#globalSuccessRate {
  text-align: center;
  font-size: 100px;
}

#ratesPerWorkflow {
  width: 80%;
  margin: 0 auto;
}

#ratesPerWorkflow .card-body div {
  margin-top: 20px;
  padding-left: 80px;
}

#ratesPerWorkflow table {
  width: 100%;
}

#ratesPerWorkflow .left {
  font-size: 35px;
}

#ratesPerWorkflow .right {
  font-size: 40px;
}

.subSection {
  margin-top:50px;
}

.messagesWrapper {
  position: relative;
  margin-top: 20px;
  padding-right: 10px;
}

#divWorkflowMessages {
  display:grid;
  grid-template-columns:1fr 1fr 1fr;
}`

/* vim: set tabstop=4:softtabstop=4:shiftwidth=4:expandtab */
