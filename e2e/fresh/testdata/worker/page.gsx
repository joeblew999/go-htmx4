package main

component Page(visits int, env string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<title>Fresh Worker</title>
		</head>
		<body>
			<h1>Fresh Worker</h1>
			<p id="env">{ env }</p>
			<Count n={visits}/>
		</body>
	</html>
}

component Count(n int) {
	<p id="visits">{ n } visits</p>
}
