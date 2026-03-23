document.addEventListener('DOMContentLoaded', () => {
	document.querySelectorAll("form").forEach(form => {
		form.addEventListener("submit", async (e) => {
			e.preventDefault();

			const formData = new FormData(form);
			form.querySelectorAll('input[type="checkbox"]').forEach(cb => {
				if (!formData.has(cb.name)) {
					formData.append(cb.name, "false");
				} else {
					formData.set(cb.name, "true");
				}
			});
			/*for (const [key, value] of formData.entries()) {
				console.log(key, value); // debug
			}*/

			const body = new URLSearchParams(formData);
			const postURL = window.location.origin + window.location.pathname;

			try {
				disableScreenPassthrough();
				const response = await fetch(postURL, {
					method: "POST",
					body,
				});
				const unmarshalledResponse = await response.json();

				if (unmarshalledResponse.Success) {
					sendSuccessToast(unmarshalledResponse.Message);
				} else {
					sendFailureToast(unmarshalledResponse.Message);
					enableScreenPassthrough();
					return
				}
			} catch (err) {
				sendFailureToast("Network error: " + err.message);
			}
			reloadPage();
		})
	});
});