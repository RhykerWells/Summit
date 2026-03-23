document.addEventListener('DOMContentLoaded', () => {
	// Generic selection updates 
	updateRoleOptionsSingle()
	updateRoleOptionsMulti()
	updateChannelOptionsSingle()
	updateCaseTypeSearch()
});

// Global variables for tables
const rowsPerPage = 10;
let currentPage = 1;

/**
* Updates single-role dropdowns.
* Adds click event listeners to each element with the class 'dropDownRoleSingleItem'.
* When a role is clicked, the hidden input's value is updated and the label text is adjusted.
* Disabled items are ignored.
*/
function updateRoleOptionsSingle() {
	document.querySelectorAll('.dropDownRoleSingleItem').forEach(item => {
		item.addEventListener('click', () => {
			if (item.classList.contains('disabled')) {
				e.stopPropagation();
				e.preventDefault();
				return;
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
			label.textContent = displayText
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
			const container = checkbox.closest('.input-group'); // Drop down container. Makes sure we only select objects for the appropriate role select.
			const checked = Array.from(container.querySelectorAll('.dropDownRoleCheckbox:checked'));
			const ids = checked.map(c => c.value);
			const names = checked.map(c => c.nextSibling.textContent.trim());
			const hiddenInput = container.querySelector('input[type=hidden]');

			hiddenInput.value = JSON.stringify(ids);

			const label = container.querySelector('span[id$="Label"]');
			let displayText = "Select roles";
			const joined = names.join(', ');
			if (names.length > 0) {
				displayText = joined;
				if (joined.length > 30) {
					displayText = `${checked.length} Selected`;
				}
			}
			label.textContent = displayText
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
		item.addEventListener('click', () => {
			if (item.classList.contains('disabled')) {
				e.stopPropagation();
				e.preventDefault();
				return;
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
			label.textContent = displayText
		});
	});
}

/** 
* Updates case type drop down
*/
function updateCaseTypeSearch() {
	document.querySelectorAll('.caseListItem').forEach(item => {
		item.addEventListener('click', () => {
			if (item.classList.contains('disabled')) {
				e.stopPropagation();
				e.preventDefault();
				return;
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
			label.textContent = displayText

			hiddenInput.dispatchEvent(new Event("input", { bubbles: true }))
		});
	});
}

// Case system


function filterCases(tableID, noRowID, filters, resetPage = true) {
	const table  = document.getElementById(tableID);
	const rows   = table.getElementsByTagName("tr");
	const noRow = noRowID ? document.getElementById(noRowID) : null;

	// get the filter values
	const values = filters.map(f => ({
		Column: f.ColumnIndex,
		Value: document.getElementById(f.InputID).value
	}));

	// filter matches
	lastMatches = [];
	for (let i = 1; i < rows.length; i++) {
		if (rows[i].id === noRowID) continue;

		let matchesAll = true;
		for (const f of values) {
			const cell = rows[i].getElementsByTagName("td")[f.Column];
			if (!cell || !cell.textContent.includes(f.Value)) {
				matchesAll = false;
				break;
			}
		}

		if (matchesAll) lastMatches.push(rows[i]);
		else rows[i].style.display = "none";
	}

	// reset if needed
	if (resetPage) currentPage = 1;

	// pagination
	const totalPages = Math.ceil(lastMatches.length / rowsPerPage) || 1;
	if (currentPage > totalPages) currentPage = totalPages;
	const start = (currentPage - 1) * rowsPerPage;
	const end   = start + rowsPerPage;

	// hide all matches first
	lastMatches.forEach(row => row.style.display = "none");
	// show only the slice for this page
	lastMatches.slice(start, end).forEach(row => row.style.display = "");

	if (noRow) noRow.style.display = lastMatches.length === 0 ? "" : "none";

	// pagination buttons
	const paginationContainerID = tableID + "-pagination";
	let container = document.getElementById(paginationContainerID);
	if (!container) return;

	container.innerHTML = "";

	const prevBtn = document.createElement("button");
	prevBtn.textContent = "Previous";
	prevBtn.className = "btn btn-sm btn-secondary me-2";
	prevBtn.disabled = currentPage === 1;
	prevBtn.onclick = () => { currentPage--; filterCases(tableID, noRowID, filters, false); };
	container.appendChild(prevBtn);

	for (let i = 1; i <= totalPages; i++) {
		const btn = document.createElement("button");
		btn.textContent = i;
		btn.className = "btn btn-sm " + (i === currentPage ? "btn-primary" : "btn-outline-primary") + " mx-1";
		btn.style.cssText = "background-color: var(--basePurple); border: 1px solid var(--accentGrey);"
		btn.onclick = () => { currentPage = i; filterCases(tableID, noRowID, filters, false); };
		container.appendChild(btn);
	}

	const nextBtn = document.createElement("button");
	nextBtn.textContent = "Next";
	nextBtn.className = "btn btn-sm btn-secondary ms-2";
	nextBtn.disabled = currentPage === totalPages;
	nextBtn.onclick = () => { currentPage++; filterCases(tableID, noRowID, filters, false); };
	container.appendChild(nextBtn);
}

function filterResponses(tableID, noRowID, filters, resetPage = true) {
	const table  = document.getElementById(tableID);
	const rows   = table.getElementsByTagName("tr");
	const noRow = noRowID ? document.getElementById(noRowID) : null;

	// get the filter values
	const values = filters.map(f => ({
		Column: f.ColumnIndex,
		Value: document.getElementById(f.InputID).value
	}));

	// filter matches
	lastMatches = [];
	for (let i = 1; i < rows.length; i++) {
		if (rows[i].id === noRowID) continue;
		
		let matchesAll = true;
		for (const f of values) {
			const cell = rows[i].getElementsByTagName("td")[f.Column];
			if (!cell || !cell.textContent.includes(f.Value)) {
				matchesAll = false;
				break;
			}
		}

		if (matchesAll) lastMatches.push(rows[i]);
		else rows[i].style.display = "none";
	}

	// reset if needed
	if (resetPage) currentPage = 1;

	// pagination
	const totalPages = Math.ceil(lastMatches.length / rowsPerPage) || 1;
	if (currentPage > totalPages) currentPage = totalPages;
	const start = (currentPage - 1) * rowsPerPage;
	const end   = start + rowsPerPage;

	// hide all matches first
	lastMatches.forEach(row => row.style.display = "none");
	// show only the slice for this page
	lastMatches.slice(start, end).forEach(row => row.style.display = "");

	if (noRow) noRow.style.display = lastMatches.length === 0 ? "" : "none";

	// pagination buttons
	const paginationContainerID = tableID + "-pagination";
	let container = document.getElementById(paginationContainerID);
	if (!container) return;

	container.innerHTML = "";

	const prevBtn = document.createElement("button");
	prevBtn.textContent = "Previous";
	prevBtn.className = "btn btn-sm btn-secondary me-2";
	prevBtn.disabled = currentPage === 1;
	prevBtn.onclick = () => { currentPage--; filterResponses(tableID, noRowID, filters, false); };
	container.appendChild(prevBtn);

	for (let i = 1; i <= totalPages; i++) {
		const btn = document.createElement("button");
		btn.textContent = i;
		btn.className = "btn btn-sm " + (i === currentPage ? "btn-primary" : "btn-outline-primary") + " mx-1";
		btn.style.cssText = "background-color: var(--basePurple); border: 1px solid var(--accentGrey);"
		btn.onclick = () => { currentPage = i; filterResponses(tableID, noRowID, filters, false); };
		container.appendChild(btn);
	}

	const nextBtn = document.createElement("button");
	nextBtn.textContent = "Next";
	nextBtn.className = "btn btn-sm btn-secondary ms-2";
	nextBtn.disabled = currentPage === totalPages;
	nextBtn.onclick = () => { currentPage++; filterResponses(tableID, noRowID, filters, false); };
	container.appendChild(nextBtn);
}

function filterShop(tableID, noRowID, filters, resetPage = true) {
	const table  = document.getElementById(tableID);
	const rows   = table.getElementsByTagName("tr");
	const noRow = noRowID ? document.getElementById(noRowID) : null;

	// get the filter values
	const values = filters.map(f => ({
		Column: f.ColumnIndex,
		Value: document.getElementById(f.InputID).value
	}));

	// filter matches
	lastMatches = [];
	for (let i = 1; i < rows.length; i++) {
		if (rows[i].id === noRowID) continue;

		let matchesAll = true;
		for (const f of values) {
			const cell = rows[i].getElementsByTagName("td")[f.Column];
			if (!cell || !cell.textContent.includes(f.Value)) {
				matchesAll = false;
				break;
			}
		}

		if (matchesAll) lastMatches.push(rows[i]);
		else rows[i].style.display = "none";
	}

	// reset if needed
	if (resetPage) currentPage = 1;

	// pagination
	const totalPages = Math.ceil(lastMatches.length / rowsPerPage) || 1;
	if (currentPage > totalPages) currentPage = totalPages;
	const start = (currentPage - 1) * rowsPerPage;
	const end   = start + rowsPerPage;

	// hide all matches first
	lastMatches.forEach(row => row.style.display = "none");
	// show only the slice for this page
	lastMatches.slice(start, end).forEach(row => row.style.display = "");

	if (noRow) noRow.style.display = lastMatches.length === 0 ? "" : "none";

	// pagination buttons
	const paginationContainerID = tableID + "-pagination";
	let container = document.getElementById(paginationContainerID);
	if (!container) return;

	container.innerHTML = "";

	const prevBtn = document.createElement("button");
	prevBtn.textContent = "Previous";
	prevBtn.className = "btn btn-sm btn-secondary me-2";
	prevBtn.disabled = currentPage === 1;
	prevBtn.onclick = () => { currentPage--; filterResponses(tableID, noRowID, filters, false); };
	container.appendChild(prevBtn);

	for (let i = 1; i <= totalPages; i++) {
		const btn = document.createElement("button");
		btn.textContent = i;
		btn.className = "btn btn-sm " + (i === currentPage ? "btn-primary" : "btn-outline-primary") + " mx-1";
		btn.style.cssText = "background-color: var(--basePurple); border: 1px solid var(--accentGrey);"
		btn.onclick = () => { currentPage = i; filterResponses(tableID, noRowID, filters, false); };
		container.appendChild(btn);
	}

	const nextBtn = document.createElement("button");
	nextBtn.textContent = "Next";
	nextBtn.className = "btn btn-sm btn-secondary ms-2";
	nextBtn.disabled = currentPage === totalPages;
	nextBtn.onclick = () => { currentPage++; filterResponses(tableID, noRowID, filters, false); };
	container.appendChild(nextBtn);
}
