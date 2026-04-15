(function () {
  'use strict';

  var container = document.getElementById('cy-container');
  if (!container || typeof cytoscape === 'undefined') return;

  var mode = container.dataset.mode;
  var token = window.__fresnelToken;
  var headers = { 'Accept': 'application/json' };
  if (token) headers['Authorization'] = 'Bearer ' + token;

  if (mode === 'dashboard') {
    var jsonStr = container.getAttribute('data-graph-json');
    if (!jsonStr) return;
    try {
      var data = JSON.parse(jsonStr);
      initDashboardGraph(data);
    } catch (err) {
      container.innerHTML = '<div class="graph-loading" style="color:var(--tlp-red)">Failed to parse graph data.</div>';
      console.error('Dashboard graph parse error:', err);
    }
  } else {
    var apiUrl = container.dataset.apiUrl;
    var eventId = container.dataset.eventId;

    fetch(apiUrl, { headers: headers })
      .then(function (res) { return res.json(); })
      .then(function (data) { initCorrelationGraph(data, eventId); })
      .catch(function (err) {
        container.innerHTML = '<div class="graph-loading" style="color:var(--tlp-red)">Failed to load graph data.</div>';
        console.error('Graph fetch error:', err);
      });
  }

  function impactColor(impact) {
    switch (impact) {
      case 'CRITICAL': return '#ef4444';
      case 'HIGH': return '#fb923c';
      case 'MODERATE': return '#facc15';
      case 'LOW': return '#3b82f6';
      default: return '#6b7280';
    }
  }

  function statusOpacity(status) {
    switch (status) {
      case 'OPEN': case 'INVESTIGATING': return 1.0;
      case 'MITIGATING': return 0.75;
      case 'RESOLVED': return 0.50;
      case 'CLOSED': return 0.25;
      default: return 1.0;
    }
  }

  function initCorrelationGraph(data, eventId) {
    container.innerHTML = '';

    var elements = [];

    (data.nodes || []).forEach(function (n) {
      elements.push({
        group: 'nodes',
        data: {
          id: n.id,
          label: n.title,
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
      style: graphStyles(),
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

    bindNodeTap(cy);
  }

  function initDashboardGraph(data) {
    container.innerHTML = '';

    var elements = [];

    (data.nodes || []).forEach(function (n) {
      var isCampaign = n.node_type === 'campaign';
      elements.push({
        group: 'nodes',
        data: {
          id: n.id,
          label: n.title,
          fullTitle: n.title,
          impact: n.impact,
          status: n.status,
          eventType: n.event_type,
          tlp: n.tlp,
          updatedAt: n.updated_at,
          nodeType: n.node_type || 'event',
          borderColor: isCampaign ? '#a78bfa' : impactColor(n.impact),
          fillOpacity: isCampaign ? 0.9 : statusOpacity(n.status),
          shape: isCampaign ? 'diamond' : 'ellipse'
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
          edgeType: e.edge_type || '',
          isRelationship: e.line_style === 'dotted'
        }
      });
    });

    if (elements.length === 0) {
      container.innerHTML = '<div class="graph-loading">No events to display.</div>';
      return;
    }

    var hasEdges = (data.edges || []).length > 0;
    var layout = hasEdges
      ? {
          name: 'cose',
          animate: true,
          animationDuration: 500,
          nodeRepulsion: 10000,
          idealEdgeLength: 150,
          gravity: 0.25,
          padding: 30
        }
      : {
          name: 'grid',
          rows: Math.ceil(Math.sqrt((data.nodes || []).length)),
          padding: 20,
          animate: true,
          animationDuration: 300
        };

    var cy = cytoscape({
      container: container,
      elements: elements,
      style: graphStyles(),
      layout: layout,
      minZoom: 0.1,
      maxZoom: 3,
      wheelSensitivity: 0.3
    });

    bindNodeTap(cy);
  }

  function graphStyles() {
    return [
      {
        selector: 'node',
        style: {
          'label': 'data(label)',
          'text-wrap': 'wrap',
          'text-max-width': '100px',
          'text-overflow-wrap': 'anywhere',
          'font-size': '10px',
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
          'width': 36,
          'height': 36
        }
      },
      {
        selector: 'node[?isSeed]',
        style: {
          'width': 50,
          'height': 50,
          'border-width': 4,
          'font-weight': 'bold'
        }
      },
      {
        selector: 'node[nodeType="campaign"]',
        style: {
          'shape': 'diamond',
          'width': 44,
          'height': 44,
          'background-color': '#a78bfa',
          'border-color': '#7c3aed',
          'border-width': 3,
          'font-weight': 'bold',
          'color': '#ddd6fe'
        }
      },
      {
        selector: 'edge',
        style: {
          'label': 'data(label)',
          'font-size': '9px',
          'color': '#8b949e',
          'text-wrap': 'wrap',
          'text-max-width': '80px',
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
        selector: 'edge[edgeType="campaign"]',
        style: {
          'line-color': '#7c3aed',
          'target-arrow-color': '#7c3aed',
          'line-style': 'dashed',
          'width': 1.5,
          'opacity': 0.6
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
    ];
  }

  function bindNodeTap(cy) {
    cy.on('tap', 'node', function (evt) {
      var node = evt.target;
      var id = node.data('id');
      var panel = document.getElementById('graph-panel-body');
      if (panel) {
        panel.innerHTML = '<div style="text-align:center;padding:1rem;"><span class="spinner"></span></div>';
        panel.innerHTML =
          '<div style="padding:.5rem 0;">' +
          '<strong style="word-break:break-word;">' + escapeHtml(node.data('fullTitle') || node.data('label')) + '</strong>' +
          '<div class="flex gap-sm mt-sm" style="flex-wrap:wrap;">' +
          '<span class="badge badge-impact-' + (node.data('impact') || 'info').toLowerCase() + '">' + (node.data('impact') || '—') + '</span>' +
          '<span class="badge badge-status-' + (node.data('status') || 'open').toLowerCase() + '">' + (node.data('status') || '—') + '</span>' +
          '<span class="badge badge-tlp-' + (node.data('tlp') || 'clear').toLowerCase() + '">TLP:' + (node.data('tlp') || '—') + '</span>' +
          '</div>' +
          (node.data('orgName') ? '<div class="text-sm text-muted mt-sm">' + escapeHtml(node.data('orgName')) + '</div>' : '') +
          '<div class="text-xs text-muted mt-sm">' + escapeHtml(node.data('eventType') || '') + '</div>' +
          '<div style="margin-top:.75rem;">' +
          '<a class="btn btn-sm" href="/events/' + id + '" data-event-link="' + id + '">' +
          'Open Event</a>' +
          '</div></div>';
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
