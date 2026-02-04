// Ottavia Charts - Lightweight Canvas-based chart renderer
// Supports line charts with pan/zoom, tooltips, and guide lines

(function(window) {
  'use strict';

  // Color schemes for dark/light mode
  const COLORS = {
    light: {
      background: 'rgba(255, 255, 255, 0)',
      grid: 'rgba(0, 0, 0, 0.08)',
      axis: 'rgba(0, 0, 0, 0.3)',
      text: 'rgba(0, 0, 0, 0.7)',
      guide: 'rgba(59, 130, 246, 0.4)',
      guideText: 'rgba(59, 130, 246, 0.8)',
      tooltip: 'rgba(255, 255, 255, 0.95)',
      tooltipText: '#1f2937',
      tooltipBorder: 'rgba(0, 0, 0, 0.1)',
      series: [
        'rgba(59, 130, 246, 1)',    // blue
        'rgba(239, 68, 68, 1)',     // red
        'rgba(34, 197, 94, 1)',     // green
        'rgba(249, 115, 22, 1)',    // orange
        'rgba(168, 85, 247, 1)',    // purple
      ]
    },
    dark: {
      background: 'rgba(0, 0, 0, 0)',
      grid: 'rgba(255, 255, 255, 0.08)',
      axis: 'rgba(255, 255, 255, 0.3)',
      text: 'rgba(255, 255, 255, 0.7)',
      guide: 'rgba(96, 165, 250, 0.4)',
      guideText: 'rgba(96, 165, 250, 0.8)',
      tooltip: 'rgba(31, 41, 55, 0.95)',
      tooltipText: '#f9fafb',
      tooltipBorder: 'rgba(255, 255, 255, 0.1)',
      series: [
        'rgba(96, 165, 250, 1)',    // blue
        'rgba(248, 113, 113, 1)',   // red
        'rgba(74, 222, 128, 1)',    // green
        'rgba(251, 146, 60, 1)',    // orange
        'rgba(192, 132, 252, 1)',   // purple
      ]
    }
  };

  // Chart class
  class OttaviaChart {
    constructor(canvas, options = {}) {
      this.canvas = canvas;
      this.ctx = canvas.getContext('2d');
      this.options = Object.assign({
        padding: { top: 20, right: 20, bottom: 40, left: 60 },
        xLabel: '',
        yLabel: '',
        xUnit: '',
        yUnit: '',
        logScaleX: false,
        logScaleY: false,
        minX: null,
        maxX: null,
        minY: null,
        maxY: null,
        guideLines: [],
        gridLines: true,
        animate: false
      }, options);

      this.data = { series: {} };
      this.viewState = {
        zoom: 1,
        panX: 0,
        panY: 0,
        isDragging: false,
        dragStart: null,
        lastPan: { x: 0, y: 0 }
      };

      this.tooltip = {
        visible: false,
        x: 0,
        y: 0,
        content: []
      };

      this.setupCanvas();
      this.bindEvents();
    }

    setupCanvas() {
      // Handle high DPI displays
      const rect = this.canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;

      this.canvas.width = rect.width * dpr;
      this.canvas.height = rect.height * dpr;

      this.ctx.scale(dpr, dpr);

      this.width = rect.width;
      this.height = rect.height;
    }

    getColors() {
      const isDark = document.documentElement.classList.contains('dark');
      return COLORS[isDark ? 'dark' : 'light'];
    }

    bindEvents() {
      // Mouse events for tooltip and pan
      this.canvas.addEventListener('mousemove', (e) => this.onMouseMove(e));
      this.canvas.addEventListener('mousedown', (e) => this.onMouseDown(e));
      this.canvas.addEventListener('mouseup', () => this.onMouseUp());
      this.canvas.addEventListener('mouseleave', () => this.onMouseLeave());

      // Wheel for zoom
      this.canvas.addEventListener('wheel', (e) => this.onWheel(e), { passive: false });

      // Touch events for mobile
      this.canvas.addEventListener('touchstart', (e) => this.onTouchStart(e), { passive: false });
      this.canvas.addEventListener('touchmove', (e) => this.onTouchMove(e), { passive: false });
      this.canvas.addEventListener('touchend', () => this.onMouseUp());

      // Resize observer
      this.resizeObserver = new ResizeObserver(() => {
        this.setupCanvas();
        this.render();
      });
      this.resizeObserver.observe(this.canvas);

      // Theme change observer
      this.themeObserver = new MutationObserver(() => this.render());
      this.themeObserver.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ['class']
      });
    }

    destroy() {
      this.resizeObserver?.disconnect();
      this.themeObserver?.disconnect();
    }

    setData(data) {
      this.data = data;
      this.calculateBounds();
      this.resetView();
      this.render();
    }

    calculateBounds() {
      const xData = this.data.series.x || [];
      const yKeys = Object.keys(this.data.series).filter(k => k !== 'x');

      // Calculate data bounds
      let minX = Infinity, maxX = -Infinity;
      let minY = Infinity, maxY = -Infinity;

      for (let i = 0; i < xData.length; i++) {
        const x = xData[i];
        if (isFinite(x)) {
          minX = Math.min(minX, x);
          maxX = Math.max(maxX, x);
        }
      }

      for (const key of yKeys) {
        const yData = this.data.series[key] || [];
        for (let i = 0; i < yData.length; i++) {
          const y = yData[i];
          if (isFinite(y)) {
            minY = Math.min(minY, y);
            maxY = Math.max(maxY, y);
          }
        }
      }

      // Add padding to y-axis
      const yPadding = (maxY - minY) * 0.1 || 1;
      minY -= yPadding;
      maxY += yPadding;

      // Use options if specified
      this.bounds = {
        dataMinX: minX,
        dataMaxX: maxX,
        dataMinY: minY,
        dataMaxY: maxY,
        minX: this.options.minX ?? minX,
        maxX: this.options.maxX ?? maxX,
        minY: this.options.minY ?? minY,
        maxY: this.options.maxY ?? maxY
      };
    }

    resetView() {
      this.viewState = {
        zoom: 1,
        panX: 0,
        panY: 0,
        isDragging: false,
        dragStart: null,
        lastPan: { x: 0, y: 0 }
      };
    }

    getPlotArea() {
      const { padding } = this.options;
      return {
        x: padding.left,
        y: padding.top,
        width: this.width - padding.left - padding.right,
        height: this.height - padding.top - padding.bottom
      };
    }

    getVisibleBounds() {
      const { zoom, panX, panY } = this.viewState;
      const { minX, maxX, minY, maxY } = this.bounds;

      const rangeX = (maxX - minX) / zoom;
      const rangeY = (maxY - minY) / zoom;

      const centerX = (minX + maxX) / 2 - panX;
      const centerY = (minY + maxY) / 2 - panY;

      return {
        minX: centerX - rangeX / 2,
        maxX: centerX + rangeX / 2,
        minY: centerY - rangeY / 2,
        maxY: centerY + rangeY / 2
      };
    }

    dataToCanvas(x, y) {
      const plot = this.getPlotArea();
      const visible = this.getVisibleBounds();

      let canvasX, canvasY;

      if (this.options.logScaleX && x > 0) {
        const logMin = Math.log10(Math.max(visible.minX, 1));
        const logMax = Math.log10(visible.maxX);
        canvasX = plot.x + ((Math.log10(x) - logMin) / (logMax - logMin)) * plot.width;
      } else {
        canvasX = plot.x + ((x - visible.minX) / (visible.maxX - visible.minX)) * plot.width;
      }

      canvasY = plot.y + plot.height - ((y - visible.minY) / (visible.maxY - visible.minY)) * plot.height;

      return { x: canvasX, y: canvasY };
    }

    canvasToData(canvasX, canvasY) {
      const plot = this.getPlotArea();
      const visible = this.getVisibleBounds();

      let dataX, dataY;

      const normX = (canvasX - plot.x) / plot.width;
      const normY = 1 - (canvasY - plot.y) / plot.height;

      if (this.options.logScaleX) {
        const logMin = Math.log10(Math.max(visible.minX, 1));
        const logMax = Math.log10(visible.maxX);
        dataX = Math.pow(10, logMin + normX * (logMax - logMin));
      } else {
        dataX = visible.minX + normX * (visible.maxX - visible.minX);
      }

      dataY = visible.minY + normY * (visible.maxY - visible.minY);

      return { x: dataX, y: dataY };
    }

    render() {
      const colors = this.getColors();
      const ctx = this.ctx;

      // Clear canvas
      ctx.clearRect(0, 0, this.width, this.height);

      // Draw grid
      if (this.options.gridLines) {
        this.drawGrid(colors);
      }

      // Draw axes
      this.drawAxes(colors);

      // Draw guide lines
      this.drawGuideLines(colors);

      // Draw series
      this.drawSeries(colors);

      // Draw tooltip
      if (this.tooltip.visible) {
        this.drawTooltip(colors);
      }
    }

    drawGrid(colors) {
      const ctx = this.ctx;
      const plot = this.getPlotArea();
      const visible = this.getVisibleBounds();

      ctx.strokeStyle = colors.grid;
      ctx.lineWidth = 1;

      // X grid lines
      const xTicks = this.getAxisTicks(visible.minX, visible.maxX, 6, this.options.logScaleX);
      for (const tick of xTicks) {
        const pos = this.dataToCanvas(tick, 0);
        if (pos.x >= plot.x && pos.x <= plot.x + plot.width) {
          ctx.beginPath();
          ctx.moveTo(pos.x, plot.y);
          ctx.lineTo(pos.x, plot.y + plot.height);
          ctx.stroke();
        }
      }

      // Y grid lines
      const yTicks = this.getAxisTicks(visible.minY, visible.maxY, 5, false);
      for (const tick of yTicks) {
        const pos = this.dataToCanvas(0, tick);
        if (pos.y >= plot.y && pos.y <= plot.y + plot.height) {
          ctx.beginPath();
          ctx.moveTo(plot.x, pos.y);
          ctx.lineTo(plot.x + plot.width, pos.y);
          ctx.stroke();
        }
      }
    }

    drawAxes(colors) {
      const ctx = this.ctx;
      const plot = this.getPlotArea();
      const visible = this.getVisibleBounds();

      ctx.fillStyle = colors.text;
      ctx.font = '11px system-ui, -apple-system, sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';

      // X axis labels
      const xTicks = this.getAxisTicks(visible.minX, visible.maxX, 6, this.options.logScaleX);
      for (const tick of xTicks) {
        const pos = this.dataToCanvas(tick, 0);
        if (pos.x >= plot.x && pos.x <= plot.x + plot.width) {
          ctx.fillText(this.formatAxisValue(tick, this.options.xUnit), pos.x, plot.y + plot.height + 5);
        }
      }

      // X axis label
      if (this.options.xLabel) {
        ctx.fillText(this.options.xLabel, plot.x + plot.width / 2, this.height - 10);
      }

      // Y axis labels
      ctx.textAlign = 'right';
      ctx.textBaseline = 'middle';
      const yTicks = this.getAxisTicks(visible.minY, visible.maxY, 5, false);
      for (const tick of yTicks) {
        const pos = this.dataToCanvas(0, tick);
        if (pos.y >= plot.y && pos.y <= plot.y + plot.height) {
          ctx.fillText(this.formatAxisValue(tick, this.options.yUnit), plot.x - 5, pos.y);
        }
      }

      // Y axis label (rotated)
      if (this.options.yLabel) {
        ctx.save();
        ctx.translate(15, plot.y + plot.height / 2);
        ctx.rotate(-Math.PI / 2);
        ctx.textAlign = 'center';
        ctx.fillText(this.options.yLabel, 0, 0);
        ctx.restore();
      }

      // Draw axis lines
      ctx.strokeStyle = colors.axis;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(plot.x, plot.y);
      ctx.lineTo(plot.x, plot.y + plot.height);
      ctx.lineTo(plot.x + plot.width, plot.y + plot.height);
      ctx.stroke();
    }

    drawGuideLines(colors) {
      const ctx = this.ctx;
      const plot = this.getPlotArea();
      const guides = this.options.guideLines || [];

      ctx.setLineDash([5, 5]);
      ctx.strokeStyle = colors.guide;
      ctx.lineWidth = 1;

      for (const guide of guides) {
        const pos = this.dataToCanvas(guide.value, 0);

        if (pos.x >= plot.x && pos.x <= plot.x + plot.width) {
          ctx.beginPath();
          ctx.moveTo(pos.x, plot.y);
          ctx.lineTo(pos.x, plot.y + plot.height);
          ctx.stroke();

          // Label
          if (guide.label) {
            ctx.fillStyle = colors.guideText;
            ctx.font = '10px system-ui, -apple-system, sans-serif';
            ctx.textAlign = 'center';
            ctx.fillText(guide.label, pos.x, plot.y - 5);
          }
        }
      }

      ctx.setLineDash([]);
    }

    drawSeries(colors) {
      const ctx = this.ctx;
      const plot = this.getPlotArea();
      const xData = this.data.series.x || [];
      const yKeys = Object.keys(this.data.series).filter(k => k !== 'x');

      // Clip to plot area
      ctx.save();
      ctx.beginPath();
      ctx.rect(plot.x, plot.y, plot.width, plot.height);
      ctx.clip();

      let colorIndex = 0;
      for (const key of yKeys) {
        const yData = this.data.series[key] || [];
        const color = colors.series[colorIndex % colors.series.length];

        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;
        ctx.beginPath();

        let started = false;
        for (let i = 0; i < xData.length; i++) {
          const x = xData[i];
          const y = yData[i];

          if (!isFinite(x) || !isFinite(y)) continue;

          const pos = this.dataToCanvas(x, y);

          if (!started) {
            ctx.moveTo(pos.x, pos.y);
            started = true;
          } else {
            ctx.lineTo(pos.x, pos.y);
          }
        }

        ctx.stroke();
        colorIndex++;
      }

      ctx.restore();
    }

    drawTooltip(colors) {
      const ctx = this.ctx;
      const { x, y, content } = this.tooltip;

      if (!content.length) return;

      // Calculate tooltip dimensions
      ctx.font = '12px system-ui, -apple-system, sans-serif';
      const lineHeight = 18;
      const padding = 8;
      let maxWidth = 0;

      for (const line of content) {
        const width = ctx.measureText(line).width;
        maxWidth = Math.max(maxWidth, width);
      }

      const tooltipWidth = maxWidth + padding * 2;
      const tooltipHeight = content.length * lineHeight + padding * 2;

      // Position tooltip (avoid going off-screen)
      let tooltipX = x + 15;
      let tooltipY = y - tooltipHeight / 2;

      if (tooltipX + tooltipWidth > this.width) {
        tooltipX = x - tooltipWidth - 15;
      }
      if (tooltipY < 0) {
        tooltipY = 0;
      }
      if (tooltipY + tooltipHeight > this.height) {
        tooltipY = this.height - tooltipHeight;
      }

      // Draw background
      ctx.fillStyle = colors.tooltip;
      ctx.strokeStyle = colors.tooltipBorder;
      ctx.lineWidth = 1;

      const radius = 4;
      ctx.beginPath();
      ctx.roundRect(tooltipX, tooltipY, tooltipWidth, tooltipHeight, radius);
      ctx.fill();
      ctx.stroke();

      // Draw text
      ctx.fillStyle = colors.tooltipText;
      ctx.textAlign = 'left';
      ctx.textBaseline = 'top';

      for (let i = 0; i < content.length; i++) {
        ctx.fillText(content[i], tooltipX + padding, tooltipY + padding + i * lineHeight);
      }

      // Draw crosshair
      const plot = this.getPlotArea();
      ctx.strokeStyle = colors.guide;
      ctx.setLineDash([3, 3]);
      ctx.lineWidth = 1;

      if (x >= plot.x && x <= plot.x + plot.width) {
        ctx.beginPath();
        ctx.moveTo(x, plot.y);
        ctx.lineTo(x, plot.y + plot.height);
        ctx.stroke();
      }

      if (y >= plot.y && y <= plot.y + plot.height) {
        ctx.beginPath();
        ctx.moveTo(plot.x, y);
        ctx.lineTo(plot.x + plot.width, y);
        ctx.stroke();
      }

      ctx.setLineDash([]);
    }

    getAxisTicks(min, max, count, logScale) {
      if (logScale && min > 0) {
        const logMin = Math.floor(Math.log10(min));
        const logMax = Math.ceil(Math.log10(max));
        const ticks = [];
        for (let i = logMin; i <= logMax; i++) {
          ticks.push(Math.pow(10, i));
        }
        return ticks;
      }

      const range = max - min;
      const step = this.niceNumber(range / (count - 1), true);
      const niceMin = Math.floor(min / step) * step;
      const niceMax = Math.ceil(max / step) * step;

      const ticks = [];
      for (let v = niceMin; v <= niceMax + step * 0.5; v += step) {
        ticks.push(v);
      }
      return ticks;
    }

    niceNumber(range, round) {
      const exponent = Math.floor(Math.log10(range));
      const fraction = range / Math.pow(10, exponent);
      let niceFraction;

      if (round) {
        if (fraction < 1.5) niceFraction = 1;
        else if (fraction < 3) niceFraction = 2;
        else if (fraction < 7) niceFraction = 5;
        else niceFraction = 10;
      } else {
        if (fraction <= 1) niceFraction = 1;
        else if (fraction <= 2) niceFraction = 2;
        else if (fraction <= 5) niceFraction = 5;
        else niceFraction = 10;
      }

      return niceFraction * Math.pow(10, exponent);
    }

    formatAxisValue(value, unit) {
      let formatted;
      const absValue = Math.abs(value);

      if (absValue >= 1000000) {
        formatted = (value / 1000000).toFixed(1) + 'M';
      } else if (absValue >= 1000) {
        formatted = (value / 1000).toFixed(1) + 'k';
      } else if (absValue < 0.01 && absValue !== 0) {
        formatted = value.toExponential(1);
      } else if (Number.isInteger(value)) {
        formatted = value.toString();
      } else {
        formatted = value.toFixed(1);
      }

      return unit ? `${formatted}` : formatted;
    }

    // Event handlers
    onMouseMove(e) {
      const rect = this.canvas.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const plot = this.getPlotArea();

      if (this.viewState.isDragging) {
        const dx = x - this.viewState.dragStart.x;
        const dy = y - this.viewState.dragStart.y;

        const visible = this.getVisibleBounds();
        const xRange = visible.maxX - visible.minX;
        const yRange = visible.maxY - visible.minY;

        const dataDx = (dx / plot.width) * xRange;
        const dataDy = -(dy / plot.height) * yRange;

        this.viewState.panX = this.viewState.lastPan.x + dataDx;
        this.viewState.panY = this.viewState.lastPan.y + dataDy;

        this.render();
        return;
      }

      // Update tooltip if in plot area
      if (x >= plot.x && x <= plot.x + plot.width &&
          y >= plot.y && y <= plot.y + plot.height) {
        this.updateTooltip(x, y);
      } else {
        this.tooltip.visible = false;
        this.render();
      }
    }

    onMouseDown(e) {
      const rect = this.canvas.getBoundingClientRect();
      this.viewState.isDragging = true;
      this.viewState.dragStart = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top
      };
      this.viewState.lastPan = {
        x: this.viewState.panX,
        y: this.viewState.panY
      };
      this.canvas.style.cursor = 'grabbing';
    }

    onMouseUp() {
      this.viewState.isDragging = false;
      this.canvas.style.cursor = 'crosshair';
    }

    onMouseLeave() {
      this.viewState.isDragging = false;
      this.tooltip.visible = false;
      this.canvas.style.cursor = 'crosshair';
      this.render();
    }

    onWheel(e) {
      e.preventDefault();

      const rect = this.canvas.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const plot = this.getPlotArea();

      // Only zoom if cursor is in plot area
      if (x < plot.x || x > plot.x + plot.width ||
          y < plot.y || y > plot.y + plot.height) {
        return;
      }

      const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
      const newZoom = Math.max(0.5, Math.min(20, this.viewState.zoom * zoomFactor));

      // Zoom centered on cursor
      const dataPos = this.canvasToData(x, y);
      const oldZoom = this.viewState.zoom;
      this.viewState.zoom = newZoom;

      // Adjust pan to keep cursor position fixed
      const newDataPos = this.canvasToData(x, y);
      this.viewState.panX += (dataPos.x - newDataPos.x);
      this.viewState.panY += (dataPos.y - newDataPos.y);

      this.render();
    }

    onTouchStart(e) {
      if (e.touches.length === 1) {
        e.preventDefault();
        const touch = e.touches[0];
        const rect = this.canvas.getBoundingClientRect();
        this.viewState.isDragging = true;
        this.viewState.dragStart = {
          x: touch.clientX - rect.left,
          y: touch.clientY - rect.top
        };
        this.viewState.lastPan = {
          x: this.viewState.panX,
          y: this.viewState.panY
        };
      }
    }

    onTouchMove(e) {
      if (e.touches.length === 1 && this.viewState.isDragging) {
        e.preventDefault();
        const touch = e.touches[0];
        const rect = this.canvas.getBoundingClientRect();
        const x = touch.clientX - rect.left;
        const y = touch.clientY - rect.top;

        const dx = x - this.viewState.dragStart.x;
        const dy = y - this.viewState.dragStart.y;
        const plot = this.getPlotArea();

        const visible = this.getVisibleBounds();
        const xRange = visible.maxX - visible.minX;
        const yRange = visible.maxY - visible.minY;

        const dataDx = (dx / plot.width) * xRange;
        const dataDy = -(dy / plot.height) * yRange;

        this.viewState.panX = this.viewState.lastPan.x + dataDx;
        this.viewState.panY = this.viewState.lastPan.y + dataDy;

        this.render();
      }
    }

    updateTooltip(canvasX, canvasY) {
      const dataPos = this.canvasToData(canvasX, canvasY);
      const xData = this.data.series.x || [];
      const yKeys = Object.keys(this.data.series).filter(k => k !== 'x');

      // Find nearest data point
      let nearestIdx = 0;
      let nearestDist = Infinity;

      for (let i = 0; i < xData.length; i++) {
        const dist = Math.abs(xData[i] - dataPos.x);
        if (dist < nearestDist) {
          nearestDist = dist;
          nearestIdx = i;
        }
      }

      const x = xData[nearestIdx];
      const content = [`${this.options.xUnit || 'x'}: ${this.formatTooltipValue(x)}`];

      for (const key of yKeys) {
        const yData = this.data.series[key];
        if (yData && nearestIdx < yData.length) {
          const y = yData[nearestIdx];
          const label = this.data.units?.[key] || key;
          content.push(`${key}: ${this.formatTooltipValue(y)} ${label}`);
        }
      }

      const nearestPos = this.dataToCanvas(x, 0);
      this.tooltip = {
        visible: true,
        x: nearestPos.x,
        y: canvasY,
        content
      };

      this.render();
    }

    formatTooltipValue(value) {
      if (!isFinite(value)) return '-';
      const absValue = Math.abs(value);

      if (absValue >= 1000) {
        return value.toFixed(0);
      } else if (absValue >= 1) {
        return value.toFixed(2);
      } else {
        return value.toFixed(4);
      }
    }
  }

  // Factory function
  function createLineChart(canvas, data, options = {}) {
    const chart = new OttaviaChart(canvas, options);
    if (data) {
      chart.setData(data);
    }
    canvas.style.cursor = 'crosshair';
    return chart;
  }

  // Alpine.js component for charts
  function chartComponent() {
    return {
      chart: null,
      loading: true,
      error: null,
      module: '',
      trackId: '',

      async init() {
        this.module = this.$el.dataset.module;
        this.trackId = this.$el.dataset.trackId;

        if (!this.module || !this.trackId) {
          this.error = 'Missing module or trackId';
          this.loading = false;
          return;
        }

        await this.loadData();
      },

      async loadData() {
        this.loading = true;
        this.error = null;

        try {
          const params = new URLSearchParams({
            module: this.module,
            maxPoints: '1500'
          });

          const response = await fetch(`/api/tracks/${this.trackId}/audioscan/series?${params}`);
          if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
          }

          const data = await response.json();
          // Set loading false first, then wait for DOM update before initializing chart
          this.loading = false;
          this.$nextTick(() => {
            this.initChart(data);
          });
        } catch (e) {
          console.error('Failed to load chart data:', e);
          this.error = e.message;
          this.loading = false;
        }
      },

      initChart(data) {
        const canvas = this.$refs.canvas;
        if (!canvas) return;

        const hints = data.renderHints || {};
        const options = {
          xLabel: data.units?.x || '',
          yLabel: Object.values(data.units || {}).filter(u => u !== data.units?.x)[0] || '',
          xUnit: data.units?.x || '',
          yUnit: Object.values(data.units || {}).filter(u => u !== data.units?.x)[0] || '',
          logScaleX: hints.freqScaleLog || false,
          minX: hints.minFreqHz || null,
          maxX: hints.maxFreqHz || hints.nyquistHz || null,
          minY: hints.minDb ?? hints.minLUFS ?? hints.minCorr ?? null,
          maxY: hints.maxDb ?? hints.maxLUFS ?? hints.maxCorr ?? null,
          guideLines: this.buildGuideLines(hints)
        };

        this.chart = createLineChart(canvas, data, options);
      },

      buildGuideLines(hints) {
        const guides = [];

        // Nyquist line
        if (hints.nyquistHz) {
          guides.push({ value: hints.nyquistHz, label: `Nyquist (${hints.nyquistHz / 1000}kHz)` });
        }

        // Frequency guide lines
        if (hints.guideLinesHz) {
          for (const hz of hints.guideLinesHz) {
            let label = hz >= 1000 ? `${hz / 1000}kHz` : `${hz}Hz`;
            guides.push({ value: hz, label });
          }
        }

        return guides;
      },

      resetZoom() {
        if (this.chart) {
          this.chart.resetView();
          this.chart.render();
        }
      },

      async downloadJSON() {
        try {
          const params = new URLSearchParams({
            module: this.module,
            maxPoints: '5000'
          });

          const response = await fetch(`/api/tracks/${this.trackId}/audioscan/series?${params}`);
          const data = await response.json();

          const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `${this.trackId}_${this.module}.json`;
          a.click();
          URL.revokeObjectURL(url);
        } catch (e) {
          console.error('Failed to download data:', e);
        }
      }
    };
  }

  // Export to window
  window.OttaviaCharts = {
    createLineChart,
    chartComponent,
    OttaviaChart
  };

})(window);
