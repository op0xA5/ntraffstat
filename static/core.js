"use strict";

$(function() {
	var zeroPrefix = function(n) {
		return (n > 9 ? '' : '0')+n;
	};
	var formatTime = function(f, t) {
		return f.replace(/(2006|15|(0)?[12345])/g, function(s) {
			var n;
			switch (s) {
				case '2006': return t.getFullYear(); break;
				case '15':   return (n=t.getHours())>9?n+"":"0"+n; break;
				case '1':    return t.getMonth()+1; break;
				case '01':   return (n=t.getMonth()+1)>9?n+"":"0"+n; break;
				case '2':    return t.getDate(); break;
				case '02':   return (n=t.getDate())>9?n+"":"0"+n; break;
				case '3':    return t.getHours(); break;
				case '03':   return (n=t.getHours())>9?n+"":"0"+n; break;
				case '4':    return t.getMinutes(); break;
				case '04':   return (n=t.getMinutes())>9?n+"":"0"+n; break;
				case '5':    return t.getSeconds(); break;
				case '05':   return (n=t.getSeconds())>9?n+"":"0"+n; break;
			}
			return '';
		});
	};
	var formatString = function(f, _args) {
		var args = arguments;
		return f.replace(/\$\d+/g, function(s) {
			return args[parseInt(s.substr(1))];
		});
	};
	var renderTimeSelection = function(timespan, time) {
		if (!timespan) timespan = $('#timespan').val();
		if (time) $('#date').val(formatTime('2006-01-02', time))
		var e = $('#time').empty();
		switch (timespan) {
			case "m5":
			default:
				e.attr('disabled', null);
				var minutes = time ? (time.getHours() * 60 + time.getMinutes()) : 0;
				for (var h = 0; h < 24; h++) {
					for (var m = 0; m < 60; m += 5) {
						var txt = zeroPrefix(h)+':'+zeroPrefix(m);
						var o = $('<option>').val(txt).text(txt).appendTo(e);
						if (h * 60 + m <= minutes && minutes < h * 60 + m + 5) {
							o.attr('selected', 'selected');
						}
					}
				}
				e.show();
				break;
			case "h":
				e.attr('disabled', null);
				var hours = time ? time.getHours() : 0;
				for (var h = 0; h < 24; h++) {
					var txt = zeroPrefix(h);
					var o = $('<option>').text(txt).appendTo(e);
					if (h == hours) {
						o.attr('selected', 'selected');
					}
				}
				e.show();
				break;
			case "d":
				e.attr('disabled', 'disabled');
		}
	};
	var getSelectedTime = function(format) {
		if (format) return formatTime(format, getSelectedTime());
		var txt = $('#date').val();
		switch ($('#timespan').val()) {
			case "m5":
			default:  txt += ' '+$('#time').val()+':00'; break;
			case "h": txt += ' '+$('#time').val()+':00:00'; break;
			case "d": break;
		}
		return new Date(txt);
	};
	renderTimeSelection(false, new Date());

	var reportItem = function(name, req, body) {
		this.name = name;
		this.req = req;
		this.body = body;
		this.bodypreq = body / req;
	};
	var processReport = function (data, sortBy, desc, from, limit, filter) {
		if (sortBy && typeof sortBy != 'function') {

			switch (sortBy + '') {
				case 'name':
				default: sortBy = function(a, b) { return a.name == b.name ? 0 : (a.name > b.name ? 1 : -1); }; break;
				case 'req': sortBy = function(a, b) { return a.req == b.req ? 0 : (a.req > b.req ? 1 : -1); }; break;
				case 'body': sortBy = function(a, b) { return a.body == b.body ? 0 : (a.body > b.body ? 1 : -1); }; break;
				case 'bodypreq': sortBy = function(a, b) { return a.bodypreq == b.bodypreq ? 0 : (a.bodypreq > b.bodypreq ? 1 : -1); }; break;
			}
		}

		if (!data.length) return [];

		var items = new Array(data.length), n = 0;
		for (var i = 0; i < data.length; i++) {
			if (!filter || data.names[i].indexOf(filter) != -1) {
				items[n++] = new reportItem(data.names[i], data.data[i * 2], data.data[i * 2 + 1]);
			}
		}
		if (n < data.length) items = items.slice(0, n);

		if (sortBy) {
			if (typeof sortBy == 'string') {
				sortBy = new Function('a', 'b',
					'if (a>b) return 1;if (a<b) return -1;if (a.name>b.name) return 1;if (a.name<b.name) return -1;return 0;'
						.replace(/(a|b)(?!\.)/g, '$1.'+sortBy));
			}
			if (desc) {
				items.sort(function(a, b) { return sortBy(b, a); });
			} else {				
				items.sort(sortBy);
			}
		}

		if (limit) {
			if (limit < 0) limit = 0;
			if (!from) from = 0;
			if (from >= n || limit == 0) return [];
			items = items.slice(from, from + limit);
		}
		return { total: n, items: items };
	};
	var reportSetting = {
		sortBy: 'body',
		sortDesc: true,
		page: 1,
		pageSize: 100,
		search: '',
	};
	var cachedReport = null;
	var renderReport = function(data) {
		if (!data) data = cachedReport;
		if (!data) return;
		cachedReport = data;

		if (data.length == 0) {
			$('#report').html('<div class="noItem">No Items</div>');
			return;
		}

		renderTimeSelection(false, new Date(data.begin));
		var seconds = Date.parse(data.end) - Date.parse(data.begin);
		if (seconds < 1) seconds = 1;
		var report = processReport(data, reportSetting.sortBy, reportSetting.sortDesc,
			(reportSetting.page - 1) * reportSetting.pageSize, reportSetting.pageSize,
			reportSetting.search);

		var pageTotal = Math.floor(report.total / reportSetting.pageSize);
		if (report.total % reportSetting.pageSize > 0) pageTotal++;
		if (reportSetting.page < 1) reportSetting.page = 1;
		if (reportSetting.page > pageTotal) page = pageTotal;

		$('.sort').each(function() {
			var e = $(this);
			e.children('i').remove();
			if (e.attr('sort-by') == reportSetting.sortBy) {
				e.append('<i class="caret'+(reportSetting.sortDesc?' desc':'')+'">');
			}
		});
		$('#recCount').text(report.total+' Records');
		$('#page').val(reportSetting.page).attr('max', pageTotal);
		$('#pageTotal').val(pageTotal);

		var fg = document.createDocumentFragment();
		report.items.forEach(function(item) {
			var tr = document.createElement('TR');
			var td = document.createElement('TD');
			td.className = 'name';
			td.innerText = item.name;
			tr.appendChild(td);
			td = document.createElement('TD');
			td.innerText = item.req;
			tr.appendChild(td);
			td = document.createElement('TD');
			td.innerText = (item.req / seconds).toFixed(2);
			tr.appendChild(td);
			td = document.createElement('TD');
			td.innerText = (item.body / 1024).toFixed(2);;
			tr.appendChild(td);
			td = document.createElement('TD');
			td.innerText = (item.body / 1024 / seconds).toFixed(2);
			tr.appendChild(td);
			td = document.createElement('TD');
			td.innerText = (item.bodypreq / 1024).toFixed(2);;
			tr.appendChild(td);
			fg.appendChild(tr);
		});
		$('#report').empty().append(fg);
	};

	var eCanvas = $('<canvas>').appendTo('header');
	var cachedCharts = null, chartsSpan = 0, chartTimeformat = '2016-01-02';
	var drawCharts = function(data, mx, my) {
		if (!data) data = cachedCharts;
		if (!data || !data.length) return;
		cachedCharts = data;

		var offset = eCanvas.offset();
		eCanvas.attr({ width: offset.width, height: offset.height });
		var ctx = eCanvas[0].getContext("2d");
		ctx.font = '12px monospace';

		var pWidth = offset.width / data.length, pWidth_2 = pWidth / 2;
		var maxReq = 0, maxBody = 0;
		var items = data.data;
		for (var i = 0; i < data.length; i++) {
			if (items[i*2] > maxReq) maxReq = items[i*2];
			if (items[i*2+1] > maxBody) maxBody = items[i*2+1];
		}
		var scaleReq = offset.height / maxReq, scaleBody = offset.height / maxBody;

		var x = 0, y = 0;
		var selectedInfo = null, sWidth = 0;
		if (typeof mx == 'number' && chartsSpan) {
			sWidth = pWidth * chartsSpan
			x = Math.floor(mx / sWidth);
			ctx.fillStyle = "#B7E1F3";
			ctx.fillRect(x * sWidth, 0, sWidth, offset.height);

			var begin = new Date(data.begin);
			var time = begin.getTime();
			time += (new Date(data.end).getTime() - time) / data.length * x *chartsSpan;
			begin.setTime(time);
			selectedInfo = {
				time: begin,
			};
		}

		ctx.strokeStyle = "#25A5E4";
		ctx.lineWidth = 1;
		ctx.lineCap = "round";
		ctx.fillStyle = "#999";

		x = 0;
		ctx.beginPath();
		for (var i = 0; i < data.length; i++) {
			if (maxBody > 0) {
				y = items[i*2+1] * scaleBody;
				ctx.fillRect(x, offset.height - y, pWidth, y);
			}

			if (maxReq > 0) {
				y = items[i*2] * scaleReq;
				if (i == 0) {
					ctx.moveTo(x + pWidth_2, offset.height - y);
				} else {
					ctx.lineTo(x + pWidth_2, offset.height - y);
				}
			}

			x += pWidth;
		}
		ctx.stroke();

		if (typeof my == 'number') {
			ctx.strokeStyle = "#B7E1F3";
			ctx.lineWidth = 0.8;
			ctx.beginPath();
			ctx.moveTo(0, my);
			ctx.lineTo(offset.width, my);
			ctx.stroke();
			if (sWidth) {
				ctx.fillStyle = "#666";
				var v = 1 - my / offset.height;
				var text1, text2;
				text1 = formatString('Req $1, Body $2K', (v * maxReq).toFixed(0), (v * maxBody / 1024).toFixed(2));
				if (chartTimeformat && selectedInfo) text2 = formatTime(chartTimeformat, selectedInfo.time);
				if (mx < offset.width / 2) {
					x = Math.floor(mx / sWidth) * sWidth + sWidth + 6;
					ctx.textAlign = 'left';
				} else {
					x = Math.floor(mx / sWidth) * sWidth - 6;
					ctx.textAlign = 'right';
				}

				if (my < 16) {
					ctx.fillText(text1, x, my + 12);
					ctx.fillText(text2, x, my + 28);
				} else if (my > offset.height - 16) {
					ctx.fillText(text1, x, my - 22);
					ctx.fillText(text2, x, my - 6);
				} else {
					ctx.fillText(text1, x, my - 6);
					ctx.fillText(text2, x, my + 12);
				}
			}
		}

		return selectedInfo;
	};

	eCanvas.on('mousemove', function(evt) {
		drawCharts(null, evt.offsetX, evt.offsetY);
	});
	eCanvas.on('mouseout', function() {
		drawCharts();
	});
	eCanvas.on('click', function(evt) {
		var select = drawCharts(null, evt.offsetX, evt.offsetY);
		if (select) {
			renderTimeSelection(false, select.time);
			$('#show').click();
		}
	});

	var renderCharts = function () {
		var time = getSelectedTime(), timespan = $('#timespan').val(), timeFormat;
		switch (timespan) {
			case "m5":
			default:  timespan = 'm'; timeFormat = '2006010203'; chartsSpan = 5; chartTimeformat = '2006-01-02 03:04'; break;
			case "h": timespan = 'm'; timeFormat = '2006010203'; chartsSpan = 60; chartTimeformat = '2006-01-02 03'; break;
			case "d": timespan = 'h'; timeFormat = '20060102'; chartsSpan = 24; chartTimeformat = '2006-01-02'; break;
		}
		$.ajax({
			url: formatString('/trend/$1/$2', timespan, formatTime(timeFormat, time)),
			cache: false,
			dataType: 'json',
			success: function(data) {
				drawCharts(data);
			},
		});
	};
	renderCharts();
	setInterval(renderCharts, 60000);

	$('#timespan').change(function() {
		renderTimeSelection($(this).val());
		renderCharts();
	});
	$('#date').change(function() {
		renderCharts();
	});
	$('#now').click(function() {
		$.ajax({
			url: formatString('/table/$1/$2/realtime', $('#timespan').val(), $('#table').val()),
			cache: false,
			dataType: 'json',
			success: renderReport,
		});
	});
	$('#show').click(function() {
		var time = getSelectedTime(), timespan = $('#timespan').val(), timeFormat;
		switch (timespan) {
			case "m5":
			default:  timeFormat = '200601020304'; break;
			case "h": timeFormat = '2006010203'; break;
			case "d": timeFormat = '20060102'; break;
		}
		$.ajax({
			url: formatString('/table/$1/$2/$3', timespan, $('#table').val(), formatTime(timeFormat, time)),
			cache: false,
			dataType: 'json',
			success: renderReport,
		});
	});
	$('.sort').click(function() {
		var sortBy = $(this).attr('sort-by');
		if (!sortBy) return;

		if (sortBy == reportSetting.sortBy) reportSetting.sortDesc = !reportSetting.sortDesc;
		else {
			reportSetting.sortBy = sortBy;
			reportSetting.sortDesc = true;
		}

		renderReport();
	});
	$('#pageSize').change(function() {
		var v = parseInt($(this).val());
		if (!v) return;
		reportSetting.pageSize = v;
		renderReport();
	});
	$('#page').change(function() {
		var v = parseInt($(this).val());
		if (!v) return;
		reportSetting.page = v;
		renderReport();
	});
	$('#search').keyup(function() {
		var v = $(this).val();
		if (reportSetting.search != v) {
			reportSetting.search = v;
			reportSetting.page = 1;
			renderReport();
		}
	});
});
