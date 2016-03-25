$( ".input" ).focusin(function() {
  $( this ).find( "span" ).animate({"opacity":"0"}, 200);
});

$( ".input" ).focusout(function() {
  $( this ).find( "span" ).animate({"opacity":"1"}, 300);
});

$(".login").submit(function(){
	
	$.post( "/login", { username: $("#usernameTxt").val(), password: $("#passwordTxt").val() })
	  .done(function(data) {
		  if (data == "success") {
				  $(this).find(".submit i").removeAttr('class').addClass("fa fa-check").css({"color":"#fff"});
				  $(".submit").css({"background":"#2ecc71", "border-color":"#2ecc71"});
				  $(".feedback").show().animate({"opacity":"1", "bottom":"-80px"}, 400);
				  $("input").css({"border-color":"#2ecc71"});
				  window.location="/"
		  }else{
			  alert("Try again. "+data + " .\nClear your browser's cookies if you cannot login.")
		  }

	    
	  });
	

  return false;
});