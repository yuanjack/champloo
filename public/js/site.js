function notify_info(msg) {
    new PNotify({
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            type: 'info',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
            }
    });
}
function notify_error (msg) {
    new PNotify({
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            type: 'error',
            width: "200px",
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
                }
    });
}
function notify_success (msg) {
    new PNotify({
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            type: 'success',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
                }
    });
}
function notify(msg) {
    new PNotify({
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            width: "150px",
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
             }
    });
}