
function notify_info(msg) {
    var option = {
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            hide: true,
            type: 'info',
            width: "150px",
            icon: 'fa fa-info-circle',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
            }
    };

    show_notify(option);
}
function show_notify(option) {
    if (typeof pnotify == 'undefined') {
        pnotify = new PNotify( option);
    } else {
        pnotify.update(option);
        if (option.hide) {
            pnotify = undefined;
        }
    }
}
function notify_error (msg) {
    var option = {
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            hide: true,
            type: 'error',
            width: "150px",
            icon: 'fa fa-warning',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
                }
    };
    show_notify(option);
}
function notify_success (msg) {
    var option = {
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            hide: true,
            type: 'success',
            width: "150px",
            icon: 'fa fa-check-circle',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
                }
    };
    show_notify(option);
}
function notify_loading(msg) {
    var option = {
            text: msg,
            animate_speed: 'fast',
            stack: false,
            hide: false,
            width: "150px",
            icon: 'fa fa-cog fa-spin',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
             }
    };
    show_notify(option);
}
function notify(msg) {
    var option = {
            text: msg,
            animate_speed: 'fast',
            stack: false,
            delay: 1000,
            hide: true,
            width: "150px",
            icon: 'fa fa-exclamation-circle',
            before_open: function(PNotify){
                  PNotify.get().css({
                    "top": ($(window).height() / 2) - (PNotify.get().height() / 2) - 100,
                    "left": ($(window).width() / 2) - (PNotify.get().width() / 2)
                  });
             }
    };
    show_notify(option);
}