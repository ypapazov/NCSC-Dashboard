(function () {
  'use strict';

  var container = document.getElementById('sync-timeline-container');
  if (!container || typeof vis === 'undefined') return;

  var jsonStr = container.getAttribute('data-timeline-json');
  if (!jsonStr) {
    container.innerHTML = '<div style="padding:2rem;text-align:center;color:var(--text-muted);">No timeline data.</div>';
    return;
  }
  var raw;
  try {
    raw = JSON.parse(jsonStr);
  } catch (err) {
    container.innerHTML = '<div style="padding:2rem;text-align:center;color:var(--text-muted);">Failed to parse timeline data.</div>';
    return;
  }

  if (!raw.items || raw.items.length === 0) {
    container.innerHTML = '<div style="padding:2rem;text-align:center;color:var(--text-muted);">No events to display.</div>';
    return;
  }

  var groups = new vis.DataSet(raw.groups || []);
  var items = new vis.DataSet(raw.items || []);

  var options = {
    orientation: { axis: 'top', item: 'top' },
    verticalScroll: true,
    horizontalScroll: true,
    maxHeight: 'calc(100vh - 260px)',
    zoomKey: 'ctrlKey',
    zoomMin: 1000 * 60 * 60 * 24,
    zoomMax: 1000 * 60 * 60 * 24 * 365 * 2,
    stack: true,
    showCurrentTime: true,
    selectable: true,
    multiselect: false,
    tooltip: { followMouse: true, overflowMethod: 'flip' },
    margin: { item: { horizontal: 4, vertical: 4 } }
  };

  var timeline = new vis.Timeline(container, items, groups, options);

  timeline.on('select', function (props) {
    if (props.items.length > 0) {
      var id = props.items[0];
      htmx.ajax('GET', '/api/v1/events/' + id, { target: '#app', swap: 'innerHTML' });
      history.pushState({}, '', '/events/' + id);
    }
  });

  var fitBtn = document.getElementById('timeline-fit-btn');
  if (fitBtn) {
    fitBtn.addEventListener('click', function () {
      timeline.fit({ animation: { duration: 500, easingFunction: 'easeInOutQuad' } });
    });
  }

  var zoomSlider = document.getElementById('timeline-zoom');
  if (zoomSlider) {
    zoomSlider.addEventListener('input', function () {
      var val = parseInt(zoomSlider.value, 10);
      var minMs = options.zoomMin;
      var maxMs = options.zoomMax;
      var logMin = Math.log(minMs);
      var logMax = Math.log(maxMs);
      var target = Math.exp(logMin + (logMax - logMin) * (1 - val / 100));

      var window = timeline.getWindow();
      var center = (window.start.getTime() + window.end.getTime()) / 2;
      var half = target / 2;
      timeline.setWindow(new Date(center - half), new Date(center + half), { animation: false });
    });
  }

  var groupFilterEl = document.getElementById('timeline-group-filter');
  if (groupFilterEl) {
    var allGroups = raw.groups || [];
    var html = '';
    for (var i = 0; i < allGroups.length; i++) {
      var g = allGroups[i];
      html += '<label style="display:flex;align-items:center;gap:.35rem;padding:.2rem 0;font-size:.8rem;color:var(--text);cursor:pointer;">' +
        '<input type="checkbox" class="tl-group-cb" value="' + g.id + '" checked/> ' +
        g.content + '</label>';
    }
    groupFilterEl.innerHTML = html;

    groupFilterEl.addEventListener('change', function () {
      var cbs = groupFilterEl.querySelectorAll('.tl-group-cb');
      for (var j = 0; j < cbs.length; j++) {
        groups.update({ id: cbs[j].value, visible: cbs[j].checked });
      }
    });
  }

  setTimeout(function () {
    timeline.fit({ animation: { duration: 300, easingFunction: 'easeInOutQuad' } });
  }, 100);
})();
