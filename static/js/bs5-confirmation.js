/** Bootstrap confirmation for destructive links. */
(function($) {
  'use strict';
  var closeCurrent = null;

  $.fn.confirmationPopover = function(options) {
    var settings = $.extend({
      title: '确认操作？', btnOkText: '确认', btnCancelText: '取消',
      btnOkClass: 'btn-sm btn-danger', btnCancelClass: 'btn-sm btn-secondary',
      placement: 'left', onConfirm: function() {}, onCancel: function() {}
    }, options);

    return this.each(function() {
      var element = this;
      // DataTables can redraw the same row; bind each link only once.
      $(element).off('click.confirmation').on('click.confirmation', function(event) {
        event.preventDefault();
        event.stopPropagation();
        if (closeCurrent) closeCurrent();
        var content = '<div class="d-flex gap-2 justify-content-end">' +
          '<button type="button" class="btn ' + settings.btnCancelClass + ' confirm-cancel">' + settings.btnCancelText + '</button>' +
          '<button type="button" class="btn ' + settings.btnOkClass + ' confirm-ok">' + settings.btnOkText + '</button></div>';
        var popover = new bootstrap.Popover(element, {
          content: content, html: true, title: settings.title,
          trigger: 'manual', placement: settings.placement, sanitize: false
        });
        function close() {
          document.removeEventListener('click', outside);
          popover.dispose();
          if (closeCurrent === close) closeCurrent = null;
        }
        function outside(event) {
          var tip = document.getElementById(element.getAttribute('aria-describedby'));
          if (!tip || !tip.contains(event.target)) close();
        }
        popover.show();
        closeCurrent = close;
        var tip = document.getElementById(element.getAttribute('aria-describedby'));
        tip.querySelector('.confirm-ok').addEventListener('click', function() {
          close();
          settings.onConfirm.call(element);
          if (element.href) window.location.href = element.href;
        });
        tip.querySelector('.confirm-cancel').addEventListener('click', function() {
          close();
          settings.onCancel.call(element);
        });
        document.addEventListener('click', outside);
      });
    });
  };
})(jQuery);
