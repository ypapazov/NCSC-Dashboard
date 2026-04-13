(function () {
  'use strict';

  var selectionActive = false;

  window.__fresnelStartCampaignSelection = function () {
    selectionActive = true;
    var bar = document.getElementById('campaign-selection-bar');
    var startBtn = document.getElementById('start-campaign-btn');
    var formInline = document.getElementById('campaign-form-inline');
    if (bar) bar.style.display = '';
    if (startBtn) startBtn.style.display = 'none';
    if (formInline) formInline.style.display = 'none';
    toggleCheckboxColumns(true);
    updateCount();
  };

  window.__fresnelCancelCampaignSelection = function () {
    selectionActive = false;
    var bar = document.getElementById('campaign-selection-bar');
    var startBtn = document.getElementById('start-campaign-btn');
    if (bar) bar.style.display = 'none';
    if (startBtn) startBtn.style.display = '';
    toggleCheckboxColumns(false);
    uncheckAll();
  };

  window.__fresnelShowCampaignForm = function () {
    var formInline = document.getElementById('campaign-form-inline');
    if (formInline) formInline.style.display = '';
    var titleInput = document.getElementById('cs-title');
    if (titleInput) titleInput.focus();
  };

  window.__fresnelUpdateSelectionCount = function () {
    updateCount();
  };

  window.__fresnelSubmitCampaign = function () {
    var title = (document.getElementById('cs-title') || {}).value || '';
    var description = (document.getElementById('cs-description') || {}).value || '';
    var tlp = (document.getElementById('cs-tlp') || {}).value || 'GREEN';

    if (!title.trim()) {
      alert('Please enter a campaign title.');
      return;
    }

    var ids = getSelectedIds();
    if (ids.length === 0) {
      alert('Please select at least one event.');
      return;
    }

    var spinner = document.getElementById('campaign-create-spinner');
    var submitBtn = document.getElementById('submit-campaign-btn');
    if (spinner) spinner.style.display = 'inline-block';
    if (submitBtn) submitBtn.disabled = true;

    var token = window.__fresnelToken;
    var headers = { 'Content-Type': 'application/json', 'Accept': 'text/html' };
    if (token) headers['Authorization'] = 'Bearer ' + token;

    fetch('/api/v1/campaigns/from-selection', {
      method: 'POST',
      headers: headers,
      body: JSON.stringify({
        title: title,
        description: description,
        tlp: tlp,
        event_ids: ids
      })
    }).then(function (res) {
      if (spinner) spinner.style.display = '';
      if (submitBtn) submitBtn.disabled = false;
      var redirect = res.headers.get('HX-Redirect');
      if (redirect) {
        htmx.ajax('GET', '/api/v1' + redirect, { target: '#app', swap: 'innerHTML' });
        history.pushState({}, '', redirect);
      } else if (res.ok) {
        return res.json().then(function (data) {
          if (data && data.id) {
            htmx.ajax('GET', '/api/v1/campaigns/' + data.id, { target: '#app', swap: 'innerHTML' });
            history.pushState({}, '', '/campaigns/' + data.id);
          }
        });
      } else {
        return res.text().then(function (text) { alert('Error: ' + text); });
      }
    }).catch(function (err) {
      if (spinner) spinner.style.display = '';
      if (submitBtn) submitBtn.disabled = false;
      alert('Request failed: ' + err.message);
    });
  };

  function toggleCheckboxColumns(show) {
    var cells = document.querySelectorAll('.col-select');
    for (var i = 0; i < cells.length; i++) {
      cells[i].style.display = show ? '' : 'none';
    }
  }

  function uncheckAll() {
    var cbs = document.querySelectorAll('.event-select-cb');
    for (var i = 0; i < cbs.length; i++) cbs[i].checked = false;
    var all = document.getElementById('select-all-events');
    if (all) all.checked = false;
  }

  function getSelectedIds() {
    var cbs = document.querySelectorAll('.event-select-cb:checked');
    var ids = [];
    for (var i = 0; i < cbs.length; i++) ids.push(cbs[i].value);
    return ids;
  }

  function updateCount() {
    var ids = getSelectedIds();
    var countEl = document.getElementById('campaign-sel-count');
    var finishBtn = document.getElementById('finish-campaign-btn');
    if (countEl) countEl.textContent = ids.length + ' event' + (ids.length !== 1 ? 's' : '') + ' selected';
    if (finishBtn) finishBtn.disabled = ids.length === 0;
  }

  var selectAll = document.getElementById('select-all-events');
  if (selectAll) {
    selectAll.addEventListener('change', function () {
      var cbs = document.querySelectorAll('.event-select-cb');
      for (var i = 0; i < cbs.length; i++) cbs[i].checked = selectAll.checked;
      updateCount();
    });
  }
})();
