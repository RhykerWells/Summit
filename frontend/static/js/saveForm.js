document.addEventListener('DOMContentLoaded', () => {
	document.querySelectorAll("form").forEach(form => {
		form.addEventListener("submit", async (e) => {
			e.preventDefault();

			const formData = new FormData(form);

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