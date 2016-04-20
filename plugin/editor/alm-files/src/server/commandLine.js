"use strict";
var minimist = require('minimist');
var utils = require("../common/utils");
var workingDir = require("./disk/workingDir");
var fsu = require("./utils/fsu");
var chalk = require("chalk");
exports.defaultPort = process.env.PORT /* the port by Windows azure */
    || 4444;
var defaultHost = '0.0.0.0';
var minimistOpts = {
    string: ['dir', 'config', 'host', 'httpskey', 'httpscert', 'auth'],
    boolean: ['open', 'safe', 'init', 'build'],
    alias: {
        't': ['port'],
        'd': ['dir'],
        'o': ['open'],
        'p': ['project'],
        'i': ['init'],
        'b': ['build'],
        'h': ['host'],
        'a': ['auth'],
		'k': ['httpskey'],
        'c': ['httpscert']
    },
    default: {
        t: exports.defaultPort,
        d: process.cwd(),
        o: true,
        h: defaultHost
    }
};
var argv = minimist(process.argv.slice(2), minimistOpts);
exports.getOptions = utils.once(function () {
    protectAgainstLongStringsWithSingleDash();
    var options = {
        port: argv.t,
        dir: argv.d,
        open: argv.o,
        safe: argv.safe,
        project: argv.p,
        init: argv.i,
        build: argv.b,
        filePaths: [],
        host: argv.h,
        httpskey: argv.httpskey,
        httpscert: argv.httpscert,
        auth: argv.auth,
    };
	
	for (var k in options) {
		var val = options[k]
		if (typeof (val) == "string") {
            var idxSpace = val.lastIndexOf(" ")
            if (idxSpace > 0) { // if has space inside it
                val = val.split(" ")[1]
            } else if (idxSpace === 0) { //if start's with space
                val = val.substring(1)
            }
            options[k] = val
        }

    }
	
	if (options.port && typeof(options.port) === "string") {// port is passed as string from external process/though code
        options.port = parseInt(options.port)
    }
	
    if (typeof options.port !== 'number') {
        options.port = exports.defaultPort;
    }
    if (argv.d) {
        options.dir = workingDir.makeAbsoluteIfNeeded(argv.d);
        workingDir.setProjectRoot(options.dir);
    }
    if (argv._ && argv._.length) {
        options.filePaths = argv._.map(function (x) { return workingDir.makeAbsoluteIfNeeded(x); });
    }
    // Common usage user does `alm ./srcFolder`
    // So if there was only one filePath detected and its a dir ... user probably meant `-d`
    if (options.filePaths.length == 1) {
        var filePath = workingDir.makeAbsoluteIfNeeded(options.filePaths[0]);
        if (fsu.isDir(filePath)) {
            workingDir.setProjectRoot(filePath);
            options.filePaths = [];
        }
    }
    if (options.safe) {
        console.log('---SAFE MODE---');
    }
    if (options.init && options.project) {
        console.log(chalk.red('The project option is ignored if you specific --init'));
    }
    if (options.project) {
        options.project = workingDir.makeAbsoluteIfNeeded(options.project);
        if (!options.project.endsWith('.json')) {
            options.project = options.project + '/' + 'tsconfig.json';
        }
        console.log('TSCONFIG: ', options.project);
    }
    if (options.httpskey) {
        options.httpskey = workingDir.makeAbsoluteIfNeeded(options.httpskey);
    }
    if (options.httpscert) {
        options.httpscert = workingDir.makeAbsoluteIfNeeded(options.httpscert);
    }
    return options;
});
/**
 * E.g. the user does `-user` instead of `--user`
 */
function protectAgainstLongStringsWithSingleDash() {
    var singleDashMatchers = minimistOpts.string.concat(minimistOpts.boolean)
        .map(function (x) { return '-' + x; });
    var args = process.argv.slice(2);
    var didUserTypeWithJustOneDash = args.filter(function (arg) { return singleDashMatchers.some(function (ss) { return ss == arg; }); });
    if (didUserTypeWithJustOneDash.length) {
        console.log(chalk.red('You provided the following arguments with a single dash (-foo). You probably meant to provide double dashes (--foo)'), didUserTypeWithJustOneDash);
        process.exit(1);
    }
}
