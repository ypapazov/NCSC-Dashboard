(function () {
  'use strict';

  var container = document.getElementById('cy-container');
  if (!container || typeof cytoscape === 'undefined') return;

  var apiUrl = container.dataset.apiUrl;
  var eventId = container.dataset.eventId;
  var token = window.__fresnelToken;

  var headers = { 'Accept': 'application/json' };
  if (token) headers['Authorization'] = 'Bearer ' + token;

  fetch(apiUrl, { headers: headers })
    .then(function (res) { return res.json(); })
    .then(function (data) { initGraph(data); })
    .catch(function (err) {
      container.innerHTML = '<div class="graph-loading" style="color:var(--tlp-red)">Failed to load graph data.</div>';
      console.error('Graph fetch error:', err);
    });

  function initGraph(data) {
    container.innerHTML = '';

    var elements = [];

    (data.nodes || []).forEach(function (n) {
      elements.push({
        group: 'nodes',
        data: {
          id: n.id,
          label: n.title.length > 40 ? n.title.substring(0, 37) + '...' : n.title,
          fullTitle: n.title,
          impact: n.impact,
          status: n.status,
          eventType: n.event_type,
          tlp: n.tlp,
          orgName: n.org_name || '',
          updatedAt: n.updated_at,
          shape: n.shape,
          borderColor: n.border_color,
          fillOpacity: n.fill_opacity,
          isSeed: n.id === eventId
        }
      });
    });

    (data.edges || []).forEach(function (e) {
      elements.push({
        group: 'edges',
        data: {
          id: e.id,
          source: e.source,
          target: e.target,
          label: e.label,
          lineStyle: e.line_style || 'solid',
          isRelationship: e.is_relationship
        }
      });
    });

    if (elements.length === 0) {
      container.innerHTML = '<div class="graph-loading">No correlations found for this event.</div>';
      return;
    }

    var cy = cytoscape({
      container: container,
      elements: elements,
      style: [
        {
          selector: 'node',
          style: {
            'label': 'data(label)',
            'text-wrap': 'wrap',
            'text-max-width': '120px',
            'font-size': '11px',
            'color': '#e6edf3',
            'text-valign': 'bottom',
            'text-margin-y': 6,
            'background-color': function (ele) {
              return ele.data('borderColor') || '#6b7280';
            },
            'background-opacity': function (ele) {
              return ele.data('fillOpacity') || 1;
            },
            'border-width': 3,
            'border-color': function (ele) {
              return ele.data('borderColor') || '#6b7280';
            },
            'shape': function (ele) {
              return ele.data('shape') || 'ellipse';
            },
            'width': 40,
            'height': 40
          }
        },
        {
          selector: 'node[?isSeed]',
          style: {
            'width': 55,
            'height': 55,
            'border-width': 4,
            'font-weight': 'bold'
          }
        },
        {
          selector: 'edge',
          style: {
            'label': 'data(label)',
            'font-size': '9px',
            'color': '#8b949e',
            'text-rotation': 'autorotate',
            'text-margin-y': -8,
            'curve-style': 'bezier',
            'target-arrow-shape': function (ele) {
              return ele.data('isRelationship') ? 'triangle' : 'none';
            },
            'line-style': function (ele) {
              return ele.data('lineStyle') || 'solid';
            },
            'line-color': '#30363d',
            'target-arrow-color': '#30363d',
            'width': function (ele) {
              return ele.data('lineStyle') === 'solid' ? 2 : 1.5;
            }
          }
        },
        {
          selector: 'node:active, node:selected',
          style: {
            'border-color': '#e94560',
            'border-width': 4,
            'overlay-opacity': 0.1,
            'overlay-color': '#e94560'
          }
        }
      ],
      layout: {
        name: 'cose',
        animate: true,
        animationDuration: 500,
        nodeRepulsion: 8000,
        idealEdgeLength: 120,
        gravity: 0.3,
        padding: 30
      },
      minZoom: 0.3,
      maxZoom: 3,
      wheelSensitivity: 0.3
    });

    cy.on('tap', 'node', function (evt) {
      var node = evt.target;
      var id = node.data('id');
      var panel = document.getElementById('graph-panel-body');
      if (panel) {
        panel.innerHTML = '<div style="text-align:center;padding:1rem;"><span class="spinner"></span></div>';
        var url = '/api/v1/events/' + id;
        var h = { 'Accept': 'text/html' };
        if (token) h['Authorization'] = 'Bearer ' + token;
        fetch(url, { headers: h })
          .then(function (res) { return res.text(); })
          .then(function (html) {
            panel.innerHTML =
              '<div style="padding:.5rem 0;">' +
              '<strong>' + escapeHtml(node.data('fullTitle')) + '</strong>' +
              '<div class="flex gap-sm mt-sm" style="flex-wrap:wrap;">' +
              '<span class="badge badge-impact-' + node.data('impact').toLowerCase() + '">' + node.data('impact') + '</span>' +
              '<span class="badge badge-status-' + node.data('status').toLowerCase() + '">' + node.data('status') + '</span>' +
              '<span class="badge badge-tlp-' + node.data('tlp').toLowerCase() + '">TLP:' + node.data('tlp') + '</span>' +
              '</div>' +
              '<div class="text-sm text-muted mt-sm">' + node.data('orgName') + '</div>' +
              '<div class="text-xs text-muted mt-sm">' + node.data('eventType') + '</div>' +
              '<div style="margin-top:.75rem;">' +
              '<a class="btn btn-sm" href="/events/' + id + '" ' +
              'onclick="htmx.ajax(\'GET\', \'/api/v1/events/' + id + '\', {target:\'#app\',swap:\'innerHTML\'});history.pushState({},\'\',\'/events/' + id + '\');return false;">' +
              'Open Event</a>' +
              '</div></div>';
          })
          .catch(function () {
            panel.innerHTML = '<div class="side-panel-empty">Failed to load event details.</div>';
          });
      }
    });

    cy.on('tap', function (evt) {
      if (evt.target === cy) {
        var panel = document.getElementById('graph-panel-body');
        if (panel) {
          panel.innerHTML = '<div class="side-panel-empty">Click a node to view event details.</div>';
        }
      }
    });
  }

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
})();
