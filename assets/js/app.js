const vm = new Vue({
	el: "#app",
	delimiters: ["${", "}"],
	data: {
		newUsername: "",
		response: "",
	},
	methods: {
		createNewUser(e) {
			e.preventDefault();

			fetch("/api/exercise/new-user", {
				method: "POST",
				body: JSON.stringify({
					username: this.newUsername
				}),
				headers: {
					"Content-Type": "application/json"
				}
			})
			.then(res => {
				if (res.ok) {
					return res.json();	
				}

				if (res.status === 409) {
					throw new Error("Username already taken");
				}
			})
			.then(data => {
				console.log(data);
				this.response = `User Created! username:${data.username} id: ${data._id}`
			})
			.catch(err => {
				this.response = "Username is already taken!"
			})
		}
	}
})