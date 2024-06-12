const express = require('express')
const app = express()

app.use((req,res,next) =>{
  req.time = new Date(Date.now()).toString();
  console.log(req.method,req.hostname, req.path, req.time);
  next();
});

app.get('/', function handler (req, res) {
  const fedModules = process.env.FED_MODULES
  res.send({ hello: 'world', fedModules: JSON.parse(fedModules ?? {}) })
})

// Run the server!
app.listen(3000, function () {
  console.log('Example app listening on port 3000!')
})
