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

	var reportItem = function(id, name, req, body) {
		this.id = id;
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
				items[n++] = new reportItem(i, data.names[i], data.data[i * 2], data.data[i * 2 + 1]);
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
	var cachedReport = null, reportHasInfo = false;
	var renderReport = function(data) {
		if (!data) data = cachedReport;
		if (!data) return;
		cachedReport = data;

		if (!data.length) {
			$('#report').html('<div class="noItem">No Items</div>');
			return;
		}

		renderTimeSelection(false, new Date(data.begin));
		var seconds = Math.floor((new Date(data.end).getTime() - new Date(data.begin).getTime()) / 1000);
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
			if (reportHasInfo) {
				var info = document.createElement('SPAN');
				info.innerHTML = info.className = 'info';
				info.setAttribute('data-id', item.id);
				td.appendChild(info);
			}
			tr.title = item.name;
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
			td.innerText = (item.bodypreq / 1024).toFixed(2);
			tr.appendChild(td);
			fg.appendChild(tr);
		});
		$('#report').empty().append(fg);
	};

	var eCanvas = $('<canvas>').appendTo('header');
	var cachedCharts = null, chartsSpan = 0, chartTimeformat = '2016-01-02';
	var drawCharts = function(data, mx, my) {
		if (!data) data = cachedCharts;
		if (!data) return;
		cachedCharts = data;

		var offset = eCanvas.offset();
		eCanvas.attr({ width: offset.width, height: offset.height });
		var ctx = eCanvas[0].getContext("2d");
		ctx.font = '12px monospace';
		ctx.clearRect(0, 0, offset.width, offset.height);

		if (!data.length) {
			ctx.fillStyle = '#666';
			ctx.textAlign = 'left';
			var offset = eCanvas.offset();
			ctx.clearRect(0, 0, offset.width, offset.height);
			ctx.fillText('No Charts', 26, 12);
			return null;
		}

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
			time += (new Date(data.end).getTime() - time) / data.length * x * chartsSpan;
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
				var v = 1 - my / offset.height, secs = data.duration / data.length;
				var text1, text2;
				text1 = formatString('Req $1/s, Body $2K/s', (v * maxReq / secs).toFixed(0), (v * maxBody / secs / 1024).toFixed(2));
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
			error: function() {
				drawCharts({ length: 0 });
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
			error: function() { renderReport({ length: 0 }); },
		});
	});
	$('#show').click(function() {
		var time = getSelectedTime(), timespan = $('#timespan').val(), timeFormat;
		var table = $('#table').val();
		switch (timespan) {
			case "m5":
			default:  timeFormat = '200601020304'; break;
			case "h": timeFormat = '2006010203'; break;
			case "d": timeFormat = '20060102'; break;
		}
		switch (table) {
			case "ip":   reportHasInfo = false; break;
			case "host": reportHasInfo = false; break;
			case "path": reportHasInfo = "urlinfo,path_refer"; break;
			case "url":  reportHasInfo = "urlinfo,url_refer"; break;
			case "file": reportHasInfo = "fileinfo,file_path"; break;
		}
		$.ajax({
			url: formatString('/table/$1/$2/$3', timespan, table, formatTime(timeFormat, time)),
			cache: false,
			dataType: 'json',
			success: renderReport,
			error: function() { renderReport({ length: 0 }); },
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

	var refererRender = function(type) {
		return function(e, name) {
			$('<span class="title">').text('Referers:').appendTo(e);
			var ul = $('<ul>').appendTo(e);
			$('<li class="loading">').text('loading').appendTo(ul);
			$.ajax({
				url: formatString('/refer/$1?q=$2', type, encodeURI(name)),
				cache: false,
				dataType: 'json',
				success: function(data) {
					ul.empty();
					if (!data || !data.length) {
						$('<li class="noitem">').text('No Items').appendTo(ul);
					}
					data.forEach(function(item) {
						var li = $('<li>').text(item).appendTo(ul);
						var url = item;
						if (!url.match(/^http:\/\//)) url = 'http://'+url;
						$('<a>').css({ float: 'right' }).attr({ href: url, target: '_blank', rel: 'noopener', title: url }).text('Open').appendTo(li);
					});
				},
				error: function() {
					ul.empty();
					$('<li class="failed">').text('error').appendTo(ul);
				},
			});
		}
	}
	var reportInfoRenders = {
		urlinfo: function(e, name) {
			$('<span class="title">').text('URL info:').appendTo(e);
			var url = name;
			if (!url.match(/^http:\/\//)) url = 'http://'+url;
			$('<a>').css({ float: 'right', marginLeft: 8 }).attr({ href: url, target: '_blank', rel: 'noopener', title: url }).text('Open URL').appendTo(e);			
			$('<a>').css({ float: 'right', marginLeft: 8 }).attr({ href: 'view-source:'+url, target: '_blank', rel: 'noopener', title: url }).text('View Source').appendTo(e);

			var ul = $('<ul>').appendTo(e);
			$('<li class="loading">').text('loading').appendTo(ul);
			$.ajax({
				url: formatString('/urlinfo?q=$1', encodeURI(name)),
				cache: false,
				dataType: 'json',
				success: function(data) {
					ul.empty();
					data.forEach(function(item) {
						$('<li>').text(item).appendTo(ul);
					});
				},
				error: function() {
					ul.empty();
					$('<li class="failed">').text('error').appendTo(ul);
				},
			});
		},
		path_refer: refererRender('path_refer'),
		url_refer: refererRender('url_refer'),
		fileinfo: function(e, name) {
			$('<span class="title">').text('File info:').appendTo(e);
			var ul = $('<ul>').appendTo(e);
			$('<li class="loading">').text('loading').appendTo(ul);
			$.ajax({
				url: formatString('/fileinfo?q=$1', encodeURI(name)),
				cache: false,
				dataType: 'json',
				success: function(data) {
					ul.empty();
					data.forEach(function(item) {
						$('<li>').text(item).appendTo(ul);
					});
				},
				error: function() {
					ul.empty();
					$('<li class="failed">').text('error').appendTo(ul);
				},
			});
		},
		file_path: refererRender('file_path'),
	};
	$('table').on('click', '.info', function() {
		var id = parseInt($(this).attr('data-id'));
		if (!Number.isFinite(id)) return;
		if (!(cachedReport && cachedReport.names)) return;
		var name = cachedReport.names[id];

		var body = $('#popupUrlInfo [data=body]').empty();
		$('#popupUrlInfo [data=title]').text(name);
		reportHasInfo.split(',').forEach(function(type) {
			if (reportInfoRenders.hasOwnProperty(type)) {
				var e = $('<div>').addClass(type).addClass('panel').appendTo(body);
				reportInfoRenders[type](e, cachedReport.names[id], cachedReport, id);
			}
		});
		$('#popupUrlInfo').show();
		$('body').addClass('popup-show');
	});

	$('body').on('click', '.popup .x', function() {
		$(this).closest('.popup').hide();
		$('body').removeClass('popup-show');
	});
});
