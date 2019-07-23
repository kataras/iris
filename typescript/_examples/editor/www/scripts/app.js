var User = (function () {
    function User(fullname) {
        this.name = fullname;
    }
    User.prototype.Hi = function (msg) {
        return msg + " " + this.name;
    };
    return User;
}());
var user = new User("iris web framework!");
var hi = user.Hi("Hello");
window.alert(hi);
