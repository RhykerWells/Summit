// Helper function to convert camelCase to Title Case
function camelToTitleCase(str) {
	return str
		.replace(/([A-Z])/g, ' $1')
		.replace(/^./, c => c.toUpperCase())
		.trim();
}

$('table[id]').each(function () {
	const tableObj = $(this)

	// Get metadata/label mappings
	const metaLabels = tableObj.data('metalabels') || {};

	// Filter row creation
	const columnNum = tableObj.find('thead tr:first th').length;
	let filterHTML = $(`<tr class="filter-row"></tr>`);
	for (let i = 0; i < columnNum; i++) {
		filterHTML.append(`<th></th>`);
	}
	tableObj.find('thead').append(filterHTML);

	// Expand column creation
	const hasExpandableRows = tableObj.find('tbody tr').filter(function () {
		return Object.keys($(this).data()).length > 0;
	}).length > 0;

	if (hasExpandableRows) {
		tableObj.find('thead tr').each(function () {
			$(this).prepend('<th class="expand-control"></th>')
		});

		tableObj.find('tbody tr').each(function () {
			$(this).prepend(`
				<td class="dt-control">
					<i class="fa-solid fa-caret-down"></i>
				</td>
			`)
		});
	}

	const $table = $(this).DataTable({
		paging: true,
		searching: true,
		ordering: false,

		columnDefs,

		info: true,
		lengthChange: true,
		pageLength: 10,
		lengthMenu: [10, 25, 50, 100],

		layout: {
			topStart: null,
			topEnd: null,
			bottomStart: null,
			bottomEnd: null,
			bottom: {
				pageLength: {},
				info: {},
				paging: {},
			}
		},

		initComplete: function () {
			const wrapper = $(this.api().table().container());
			const api = this.api();

			const filterCells = tableObj.find('.filter-row th')

			api.columns().every(function (colIdx) {
				const column = this;
				const header = $(column.header());
				const filterCell = filterCells.eq(colIdx);

				if (column.header().classList.contains('text-filter')) {
					$(`
						<input
							type="text"
							class="textInput form-control text-light"
							style="min-width: 4rem;"
						/>
					`)
					.appendTo(filterCell)
					.on('keyup change clear', function () {
						if (column.search() !== this.value) {
							column.search(this.value).draw();
						}
					});
				}

				if (column.header().classList.contains('dropdown-filter')) {
					let dropdown = $(`
						<div class="dropdown input-group">
							<a class="btn dropdown-toggle text-start flex-grow-1 d-flex align-items-center justify-content-between text-decoration-none text-white" role="button" data-bs-toggle="dropdown">
								<span>All</span>
								<i class="fa-solid fa-caret-down"></i>
							</a>
						</div>
					`);

					const menu = $('<ul class="dropdown-menu w-100 overflow-auto"></ul>');
					menu.append(`
						<li>
							<a href="#" class="dropdown-item" data-value="">
								All
							</a>
						</li>
					`);

					column.data().unique().sort().each(function (d) {
						const text = $('<div>').html(d).text().trim();
						if (!text) return;
						menu.append(`
							<li>
								<a class="dropdown-item" data-value="${text}">
									${text}
								</a>
							</li>
						`);
					});

					dropdown.append(menu);
					filterCell.html(dropdown);

					menu.on('click', '.dropdown-item', function (e) {
						e.preventDefault();
						const value = $(this).data('value');
						const displayText = $(this).text().trim();
						$(this)
							.closest('.dropdown')
							.find('.dropdown-toggle span')
							.text(displayText || 'All');
						column.search(value || '').draw();
					});
				}
			});

			const lengthWrapper = wrapper.find('.dt-length');
			lengthWrapper.addClass("d-flex align-items-center justify-content-between gap-1").html(`
				<div class="dropdown input-group">
					<a class="btn dropdown-toggle text-start flex-grow-1 d-flex align-items-center justify-content-between text-decoration-none text-white" role="button" data-bs-toggle="dropdown">
						<span>10</span>
						<i class="fa-solid fa-caret-down"></i>
					</a>
					<ul class="dropdown-menu w-100 overflow-auto">
						<li><a class="dropdown-item" data-value="10">10</a></li>
						<li><a class="dropdown-item" data-value="25">25</a></li>
						<li><a class="dropdown-item" data-value="50">50</a></li>
						<li><a class="dropdown-item" data-value="100">100</a></li>
					</ul>
				</div>
				<label class="text-nowrap">
					Rows per page
				</label>
			`);

			lengthWrapper.on('click', '.dropdown-item', function () {
				const value = parseInt($(this).data('value'));
				$(this)
					.closest('.dropdown')
					.find('.dropdown-toggle span')
					.text(value);
				$table.page.len(value).draw();
			});
		}
	});

	tableObj.on('click', 'tbody td.dt-control', function () {
		const tr = $(this).closest('tr');
		const row = $table.row(tr);

		if (row.child.isShown()) {
			row.child.hide();
			tr.removeClass('shown');
		} else {
			row.child(formatChildRow(tr, metaLabels)).show();
			tr.addClass('shown');
		}
	});
});

function formatChildRow(tr, metaLabels = {}) {
	const data = tr.data();
	const labelMap = metaLabels.labels || {};

	let html = '<div class="p-3">';

	Object.entries(data).forEach(([key, value]) => {
		if (key.startsWith('dt')) return;
		if (!value) return;

		// Use custom label if provided, otherwise convert camelCase to Title Case
		const displayLabel = labelMap[key] || camelToTitleCase(key);

		html += `
			<div>
				<strong>${displayLabel}:</strong> ${value}
			</div>
		`;
	});

	html += '</div>';

	return html;
}