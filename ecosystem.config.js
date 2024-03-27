
module.exports = {
  apps : [{
    name   : "bsv20",
    script : "./bsv20-nofees",
    cwd : "./cmd/bsv20-nofees",
    args : ["-s", "1600000", "-c", "1"],
    merge_logs: true,
    error_file: "output.log",
    log_file: "output.log",
    out_file: "output.log",
  }, {
    name   : "bsv20-server",
    script : "./server",
    cwd : "./cmd/server",
    merge_logs: true,
    error_file: "output.log",
    log_file: "output.log",
    out_file: "output.log",
  }]
}
