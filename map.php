<?php
	mysql_connect('localhost', 'root', '');
	mysql_select_db('outernet');

	$results = mysql_query("SELECT * FROM stats ORDER BY id DESC LIMIT 1");
	$row = mysql_fetch_assoc($results);

?>

<!DOCTYPE html>
<html>
  <head>
    <meta name="viewport" content="initial-scale=1.0, user-scalable=no">
    <meta charset="utf-8">
    <title>Receivers</title>
    <style>
      html, body {
        height: 100%;
        margin: 0;
        padding: 0;
      }
      #map {
        height: 100%;
      }
    </style>
  </head>
  <body>
    <div id="map"></div>
    <script>

	// "time since" formatting function.
	function timeSince(date) {
		var seconds = Math.floor((new Date() - date) / 1000);

		var interval = Math.floor(seconds / 31536000);

		if (interval > 1) {
			return interval + " years";
		}
		interval = Math.floor(seconds / 2592000);
		if (interval > 1) {
			return interval + " months";
		}
		interval = Math.floor(seconds / 86400);
		if (interval > 1) {
			return interval + " days";
		}
		interval = Math.floor(seconds / 3600);
		if (interval > 1) {
			return interval + " hours";
		}
		interval = Math.floor(seconds / 60);
		if (interval > 1) {
			return interval + " minutes";
		}
		return Math.floor(seconds) + " seconds";
	}

	// Parse a MySQL datetime formatted string into a 'Date' object. Assumes GMT timezone on the input.
	function parseMySQLDate(date_str) {
		// Format the GMT MySQL timestamp.
		// Split timestamp into [ Y, M, D, h, m, s ]
		var t = date_str.split(/[- :]/);
		// Apply each element to the Date function
		var date_obj = new Date(Date.UTC(t[0], t[1]-1, t[2], t[3], t[4], t[5]));
		return date_obj;
	}
	

      function initMap() {
        var pos = {lat: <?php echo $row['ReceiverLat'];?>, lng: <?php echo $row['ReceiverLng'];?>};
        var map = new google.maps.Map(document.getElementById('map'), {
          zoom: 4,
          center: pos
        });

		var lastUpdated = parseMySQLDate('<?php echo $row['TimeCollected'];?>');
        var contentString = '<h1>Receiver Status</h1><br/>'+
							'<b>SNR</b>: ' + <?php echo $row['SNR_Avg'];?> + '<br/>'+
							'<b>Data rate</b>: ' + <?php echo $row['Packets_Total'];?> + ' ppm<br/>'+
							'<b>Last updated</b>: ' + timeSince(lastUpdated) + ' ago<br/>'
							;

        var infowindow = new google.maps.InfoWindow({
          content: contentString
        });

        var marker = new google.maps.Marker({
          position: pos,
          map: map,
          title: 'cyoung'
        });
        marker.addListener('click', function() {
          infowindow.open(map, marker);
        });
      }
    </script>
    <script async defer
    src="https://maps.googleapis.com/maps/api/js?key=AIzaSyADI7QIro4kZMpSgB9UmISitcKPAVrYXyI&callback=initMap">
    </script>
  </body>
</html>