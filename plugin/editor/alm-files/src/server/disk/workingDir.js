/**
 * Controls the root directory we are working off of
 */
"use strict";
var fsu = require("../utils/fsu");
var utils = require("../../common/utils");
var projectRoot = fsu.consistentPath(process.cwd());
function getProjectRoot() {
    return projectRoot;
}
exports.getProjectRoot = getProjectRoot;
function setProjectRoot(rootDir) {
    // fix graceful-fs error when running by external process/code
	if (Object.prototype.toString.call(rootDir) === '[object Array]') {
		rootDir = rootDir[1]
	}else if (rootDir.indexOf(" ") !== -1) {
		rootDir = rootDir.split(" ")[1]
	}
    //

    projectRoot = fsu.consistentPath(rootDir);
    process.chdir(projectRoot);
}
exports.setProjectRoot = setProjectRoot;
function makeRelative(filePath) {
    return fsu.makeRelativePath(projectRoot, filePath);
}
exports.makeRelative = makeRelative;
function makeAbsolute(relativeFilePath) {
    return fsu.resolve(projectRoot, relativeFilePath);
}
exports.makeAbsolute = makeAbsolute;
function makeAbsoluteIfNeeded(filePathOrRelativeFilePath) {
    if (!fsu.isAbsolute(filePathOrRelativeFilePath)) {
        return makeAbsolute(filePathOrRelativeFilePath);
    }
    else {
        return fsu.consistentPath(filePathOrRelativeFilePath);
    }
	return filePathOrRelativeFilePath
}
exports.makeAbsoluteIfNeeded = makeAbsoluteIfNeeded;
function makeRelativeUrl(url) {
    var _a = utils.getFilePathAndProtocolFromUrl(url), filePath = _a.filePath, protocol = _a.protocol;
    var relativeFilePath = makeRelative(filePath);
    return utils.getUrlFromFilePathAndProtocol({ protocol: protocol, filePath: relativeFilePath });
}
exports.makeRelativeUrl = makeRelativeUrl;
function makeAbsoluteUrl(relativeUrl) {
    var _a = utils.getFilePathAndProtocolFromUrl(relativeUrl), relativeFilePath = _a.filePath, protocol = _a.protocol;
    var filePath = makeAbsolute(relativeFilePath);
    return utils.getUrlFromFilePathAndProtocol({ protocol: protocol, filePath: filePath });
}
exports.makeAbsoluteUrl = makeAbsoluteUrl;
