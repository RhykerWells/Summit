document.addEventListener('DOMContentLoaded', () => {
	// Generic selection updates 
	updateRoleOptionsSingle()
	updateRoleOptionsMulti()
	updateChannelOptionsSingle()
	updateCaseTypeSearch()

	// Initialize table filters
	initializeTableFilters()
});

// Global variables for tables
const DEFAULT_ROWS_PER_PAGE = 10;
const tableFilterState = {};

/**
* Updates single-role dropdowns.
* Adds click event listeners to each element with the class 'dropDownRoleSingleItem'.
* When a role is clicked, the hidden input's value is updated and the label text is adjusted.
* Disabled items are ignored.
*/
function updateRoleOptionsSingle() {
	document.querySelectorAll('.dropDownRoleSingleItem').forEach(item => {
		item.addEventListener('click', event => {
			if (item.classList.contains('disabled')) {
				event.stopPropagation();
				event.preventDefault();
			}

			const container = item.closest('.input-group'); // Drop down container. Makes sure we only select objects for the appropriate role select.
			const name = item.textContent.trim();
			const value = item.getAttribute('data-value');
			const hiddenInput = container.querySelector('input[type=hidden]');

			hiddenInput.value = value;

			const label = container.querySelector('span[id$="Label"]');
			let displayText = "Select role";
			if (value) {
				if (name.length > 30) {
					displayText = "1 Selected";
				} else {
					displayText = name;
				}
			}
			label.textContent = displayText;
		});
	});
}

/**
* Updates multi-role checkboxes.
* Adds change event listeners to each checkbox with the class 'dropDownRoleCheckbox'.
* Updates the hidden input with selected role IDs and updates the label text.
* If multiple roles are selected, the label either lists them or shows the count if the text is long.
*/
function updateRoleOptionsMulti() {
	document.querySelectorAll('.dropDownRoleCheckbox').forEach(checkbox => {
		checkbox.addEventListener('change', () => {
			const container = checkbox.closest('.input-group');

			const checked = Array.from(container.querySelectorAll('.dropDownRoleCheckbox:checked'));
			const names = checked.map(c => c.nextSibling.textContent.trim());

			const label = container.querySelector('span[id$="Label"]');

			let displayText = "Select roles";
			const joined = names.join(', ');

			if (names.length > 0) {
				displayText = joined;
				if (joined.length > 30) {
					displayText = `${checked.length} Selected`;
				}
			}

			label.textContent = displayText;
		});
	});
}

/**
* Updates single-channel dropdowns.
* Adds click event listeners to each element with the class 'channelListItem'.
* When a channel is clicked, updates the hidden input and updates the label text.
* Disabled items are ignored.
*/
function updateChannelOptionsSingle() {
	document.querySelectorAll('.channelListItem').forEach(item => {
		item.addEventListener('click', event => {
			if (item.classList.contains('disabled')) {
				event.stopPropagation();
				event.preventDefault();
			}

			const container = item.closest('.input-group'); // Drop down container. Makes sure we only select objects for the appropriate role select.
			const name = item.textContent.trim();
			const value = item.getAttribute('data-value');
			const hiddenInput = container.querySelector('input[type=hidden]');

			hiddenInput.value = value;

			const label = container.querySelector('span[id$="Label"]');
			let displayText = "Select channel";
			if (value) {
				if (name.length > 30) {
					displayText = "1 Selected";
				} else {
					displayText = name;
				}
			}
			label.textContent = displayText;
		});
	});
}

/** 
* Updates case type drop down
*/
function updateCaseTypeSearch() {
	document.querySelectorAll('.caseListItem').forEach(item => {
		item.addEventListener('click', event => {
			if (item.classList.contains('disabled')) {
				event.stopPropagation();
				event.preventDefault();
			}

			const container = item.closest('.input-group'); // Drop down container. Makes sure we only select objects for the appropriate role select.
			const name = item.textContent.trim();
			const value = item.getAttribute('data-value');
			const hiddenInput = container.querySelector('input[type=hidden]');

			hiddenInput.value = value;

			const label = container.querySelector('span[id$="Label"]');
			let displayText = "Select case type";
			if (value) {
				displayText = name;
			}
			label.textContent = displayText;

			hiddenInput.dispatchEvent(new Event("input", { bubbles: true }));
		});
	});
}

function initializeTableFilters() {
	const controls = document.querySelectorAll('[data-table-id][data-filter-column-index]');
	if (!controls.length) return;

	const tables = {};

	controls.forEach(control => {
		const tableID = control.dataset.tableId;
		if (!tableID) return;

		const columnIndex = Number(control.dataset.filterColumnIndex) || 0;
		const noRowID = control.dataset.noRowId || null;
		const rowsPerPage = Number(control.dataset.rowsPerPage) || DEFAULT_ROWS_PER_PAGE;

		if (!tables[tableID]) {
			tables[tableID] = { filters: [], noRowID, rowsPerPage };
		}

		tables[tableID].filters.push({ InputElement: control, ColumnIndex: columnIndex });
	});

	Object.entries(tables).forEach(([tableID, config]) => {
		const callback = () => filterTable(tableID, config.noRowID, config.filters, true, { rowsPerPage: config.rowsPerPage });
		config.filters.forEach(filter => {
			const eventType = filter.InputElement.tagName.toLowerCase() === 'select' ? 'change' : 'input';
			filter.InputElement.addEventListener(eventType, callback);
		});
		callback();
	});
}

function filterTable(tableID, noRowID, filters = [], resetPage = true, options = {}) {
	const table = document.getElementById(tableID);
	if (!table) return;

	const state = getTableState(tableID);
	const rowsPerPage = options.rowsPerPage || DEFAULT_ROWS_PER_PAGE;
	if (resetPage) state.currentPage = 1;

	const tbody = table.tBodies.length ? table.tBodies[0] : table;
	const rows = Array.from(tbody.querySelectorAll('tr'));

	const activeFilters = filters.map(filter => ({
		Column: typeof filter.ColumnIndex === 'number' ? filter.ColumnIndex : (typeof filter.column === 'number' ? filter.column : 0),
		Value: getFilterValue(filter)
	}));

	const visibleRows = [];

	rows.forEach(row => {
		if (row.id === noRowID) {
			row.style.display = 'none';
			return;
		}

		if (row.closest('thead')) {
			return;
		}

		const cells = Array.from(row.querySelectorAll('td'));
		const matched = activeFilters.every(filter => {
			if (!filter.Value) return true;
			const cell = cells[filter.Column];
			return cell && cell.textContent.toLowerCase().includes(filter.Value);
		});

		if (matched) {
			visibleRows.push(row);
		} else {
			row.style.display = 'none';
		}
	});

	const totalPages = Math.max(1, Math.ceil(visibleRows.length / rowsPerPage));
	if (state.currentPage > totalPages) state.currentPage = totalPages;

	const start = (state.currentPage - 1) * rowsPerPage;
	const end = start + rowsPerPage;

	visibleRows.forEach((row, index) => {
		row.style.display = index >= start && index < end ? '' : 'none';
	});

	if (noRowID) {
		const noRow = document.getElementById(noRowID);
		if (noRow) {
			noRow.style.display = visibleRows.length === 0 ? '' : 'none';
		}
	}

	renderPagination(tableID, totalPages, state.currentPage, newPage => {
		state.currentPage = newPage;
		filterTable(tableID, noRowID, filters, false, options);
	});
}

function getTableState(tableID) {
	if (!tableFilterState[tableID]) {
		tableFilterState[tableID] = { currentPage: 1 };
	}
	return tableFilterState[tableID];
}

function getFilterValue(filter) {
	if (filter.InputElement) {
		return String(filter.InputElement.value || '').trim().toLowerCase();
	}

	if (filter.InputID) {
		const input = document.getElementById(filter.InputID);
		return String(input ? input.value : '').trim().toLowerCase();
	}

	return String(filter.Value || '').trim().toLowerCase();
}

function renderPagination(tableID, totalPages, currentPage, onPageChange) {
	const paginationContainerID = `${tableID}-pagination`;
	const container = document.getElementById(paginationContainerID);
	if (!container) return;

	container.innerHTML = '';
	container.classList.add('d-flex', 'flex-wrap', 'justify-content-center', 'gap-2');

	const makeButton = (text, disabled, handler, classes = '') => {
		const btn = document.createElement('button');
		btn.type = 'button';
		btn.textContent = text;
		btn.className = `btn btn-sm ${classes}`.trim();
		btn.style.cssText = 'background-color: var(--basePurple); border: 1px solid var(--accentGrey);';
		btn.disabled = disabled;
		btn.addEventListener('click', handler);
		return btn;
	};

	container.appendChild(makeButton('Previous', currentPage === 1, () => onPageChange(currentPage - 1), 'btn-secondary'));

	for (let page = 1; page <= totalPages; page += 1) {
		container.appendChild(makeButton(String(page), false, () => onPageChange(page), page === currentPage ? 'btn-primary' : 'btn-outline-primary'));
	}

	container.appendChild(makeButton('Next', currentPage === totalPages, () => onPageChange(currentPage + 1), 'btn-secondary'));
}