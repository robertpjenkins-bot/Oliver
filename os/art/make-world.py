#!/usr/bin/env python3
"""World Explorer's map (os/profile/airootfs/usr/local/share/ollie/world.json),
made once from Natural Earth's countries (public domain, naturalearthdata.com):

  curl -fsSLO https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_110m_admin_0_countries.geojson
  python3 -I os/art/make-world.py ne_110m_admin_0_countries.geojson

Each country: its name, continent, flag (from its two-letter code), about how
many people live there, its capital and, for many, a fun fact; and its
outline, simplified (Douglas-Peucker) and rounded, for the spinning globe.
"""
import json
import math
import os
import sys

CAPITALS = {
    "AF": "Kabul", "AL": "Tirana", "DZ": "Algiers", "AO": "Luanda", "AR": "Buenos Aires", "AM": "Yerevan", "AU": "Canberra",
    "AT": "Vienna", "AZ": "Baku", "BS": "Nassau", "BD": "Dhaka", "BY": "Minsk", "BE": "Brussels", "BZ": "Belmopan",
    "BJ": "Porto-Novo", "BT": "Thimphu", "BO": "Sucre", "BA": "Sarajevo", "BW": "Gaborone", "BR": "Brasília", "BN": "Bandar Seri Begawan",
    "BG": "Sofia", "BF": "Ouagadougou", "BI": "Gitega", "KH": "Phnom Penh", "CM": "Yaoundé", "CA": "Ottawa", "CF": "Bangui",
    "TD": "N'Djamena", "CL": "Santiago", "CN": "Beijing", "CO": "Bogotá", "CG": "Brazzaville", "CD": "Kinshasa", "CR": "San José",
    "CI": "Yamoussoukro", "HR": "Zagreb", "CU": "Havana", "CY": "Nicosia", "CZ": "Prague", "DK": "Copenhagen", "DJ": "Djibouti",
    "DO": "Santo Domingo", "EC": "Quito", "EG": "Cairo", "SV": "San Salvador", "GQ": "Malabo", "ER": "Asmara", "EE": "Tallinn",
    "SZ": "Mbabane", "ET": "Addis Ababa", "FJ": "Suva", "FI": "Helsinki", "FR": "Paris", "GA": "Libreville", "GM": "Banjul",
    "GE": "Tbilisi", "DE": "Berlin", "GH": "Accra", "GR": "Athens", "GL": "Nuuk", "GT": "Guatemala City", "GN": "Conakry",
    "GW": "Bissau", "GY": "Georgetown", "HT": "Port-au-Prince", "HN": "Tegucigalpa", "HU": "Budapest", "IS": "Reykjavík",
    "IN": "New Delhi", "ID": "Jakarta", "IR": "Tehran", "IQ": "Baghdad", "IE": "Dublin", "IL": "Jerusalem", "IT": "Rome",
    "JM": "Kingston", "JP": "Tokyo", "JO": "Amman", "KZ": "Astana", "KE": "Nairobi", "KP": "Pyongyang", "KR": "Seoul",
    "KW": "Kuwait City", "KG": "Bishkek", "LA": "Vientiane", "LV": "Riga", "LB": "Beirut", "LS": "Maseru", "LR": "Monrovia",
    "LY": "Tripoli", "LT": "Vilnius", "LU": "Luxembourg", "MG": "Antananarivo", "MW": "Lilongwe", "MY": "Kuala Lumpur",
    "ML": "Bamako", "MR": "Nouakchott", "MX": "Mexico City", "MD": "Chișinău", "MN": "Ulaanbaatar", "ME": "Podgorica",
    "MA": "Rabat", "MZ": "Maputo", "MM": "Naypyidaw", "NA": "Windhoek", "NP": "Kathmandu", "NL": "Amsterdam", "NZ": "Wellington",
    "NI": "Managua", "NE": "Niamey", "NG": "Abuja", "MK": "Skopje", "NO": "Oslo", "OM": "Muscat", "PK": "Islamabad",
    "PA": "Panama City", "PG": "Port Moresby", "PY": "Asunción", "PE": "Lima", "PH": "Manila", "PL": "Warsaw", "PT": "Lisbon",
    "PR": "San Juan", "QA": "Doha", "RO": "Bucharest", "RU": "Moscow", "RW": "Kigali", "SA": "Riyadh", "SN": "Dakar",
    "RS": "Belgrade", "SL": "Freetown", "SK": "Bratislava", "SI": "Ljubljana", "SB": "Honiara", "SO": "Mogadishu",
    "ZA": "Pretoria", "SS": "Juba", "ES": "Madrid", "LK": "Sri Jayawardenepura Kotte", "SD": "Khartoum", "SR": "Paramaribo",
    "SE": "Stockholm", "CH": "Bern", "SY": "Damascus", "TW": "Taipei", "TJ": "Dushanbe", "TZ": "Dodoma", "TH": "Bangkok",
    "TL": "Dili", "TG": "Lomé", "TT": "Port of Spain", "TN": "Tunis", "TR": "Ankara", "TM": "Ashgabat", "UG": "Kampala",
    "UA": "Kyiv", "AE": "Abu Dhabi", "GB": "London", "US": "Washington, D.C.", "UY": "Montevideo", "UZ": "Tashkent",
    "VU": "Port Vila", "VE": "Caracas", "VN": "Hanoi", "YE": "Sanaa", "ZM": "Lusaka", "ZW": "Harare", "VA": "Vatican City",
    "NC": "Nouméa", "PS": "Ramallah", "EH": "Laayoune", "XK": "Pristina", "FK": "Stanley",
}
FACTS = {
    "US": "It has 50 states, and the Grand Canyon is so big you can see it from space.",
    "CA": "Canada has more lakes than all the other countries in the world put together!",
    "MX": "Chocolate and popcorn first came from Mexico.",
    "BR": "Most of the Amazon rainforest, the biggest rainforest on Earth, is in Brazil.",
    "AR": "The tango dance comes from Argentina.",
    "PE": "Machu Picchu, an old city high in the mountains, is in Peru.",
    "CL": "Chile is very long and thin: about 4,300 km long!",
    "GB": "London's Big Ben is the name of the bell, not the clock tower.",
    "IE": "Ireland is called the Emerald Isle because it is so green.",
    "FR": "The Eiffel Tower in Paris grows about 15 cm taller in summer heat.",
    "ES": "Every year, a town in Spain has a giant tomato fight called La Tomatina.",
    "PT": "Portugal makes about half of all the world's cork.",
    "IT": "Pizza was invented in Italy, in the city of Naples.",
    "DE": "Germany has more than 20,000 castles.",
    "NL": "There are more bicycles than people in the Netherlands.",
    "BE": "Belgium makes a lot of chocolate: hundreds of thousands of tonnes a year.",
    "CH": "Switzerland has more than 1,500 lakes.",
    "AT": "The world's oldest zoo still open is in Vienna, Austria.",
    "NO": "In the north of Norway, the Sun doesn't set for weeks in summer.",
    "SE": "Sweden has a hotel made of ice that is built again every winter.",
    "FI": "Finland has about 3 million saunas!",
    "DK": "LEGO bricks come from Denmark.",
    "IS": "Iceland has volcanoes, glaciers and no mosquitoes.",
    "GR": "The first Olympic Games were held in Greece, almost 3,000 years ago.",
    "PL": "Marie Curie, who won two Nobel Prizes, was born in Poland.",
    "RU": "Russia is the biggest country in the world, with 11 time zones.",
    "UA": "Ukraine is famous for its big fields of sunflowers.",
    "TR": "Istanbul, in Turkey, is a city on two continents: Europe and Asia.",
    "EG": "The pyramids of Giza in Egypt are more than 4,500 years old.",
    "MA": "Morocco has the Sahara desert and snowy mountains too.",
    "KE": "Kenya is famous for its safaris with lions, elephants and giraffes.",
    "TZ": "Mount Kilimanjaro, the tallest mountain in Africa, is in Tanzania.",
    "ZA": "South Africa has three capital cities!",
    "NG": "Nigeria has more people than any other country in Africa.",
    "MG": "Most of Madagascar's animals, like lemurs, live nowhere else on Earth.",
    "ET": "Coffee first came from Ethiopia.",
    "CN": "The Great Wall of China is thousands of kilometres long.",
    "JP": "Japan has about 6,800 islands and is home to bullet trains.",
    "KR": "South Korea is famous for K-pop music.",
    "IN": "India has more than 20 official languages.",
    "NP": "Mount Everest, the tallest mountain in the world, is on Nepal's border.",
    "TH": "Thailand is called the Land of Smiles.",
    "VN": "Vietnam has one of the biggest caves in the world: Son Doong.",
    "ID": "Indonesia is made of more than 17,000 islands.",
    "PH": "The Philippines has more than 7,000 islands.",
    "SA": "Saudi Arabia has no rivers at all.",
    "MN": "Mongolia has more horses than people.",
    "AU": "Australia has kangaroos, koalas and the Great Barrier Reef.",
    "NZ": "New Zealand has more sheep than people.",
    "GL": "Greenland is the biggest island in the world, and mostly covered in ice.",
    "AQ": "Antarctica is the coldest place on Earth, and home to penguins.",
    "CU": "Cuba is home to the bee hummingbird, the smallest bird in the world.",
    "JM": "Jamaica is where reggae music comes from.",
}
CONTINENT_EMOJI = {"Africa": "🌍", "Europe": "🌍", "Asia": "🌏", "Oceania": "🌏", "North America": "🌎",
                   "South America": "🌎", "Antarctica": "🧊", "Seven seas (open ocean)": "🌊"}


def dp(points, tol):
    """Douglas-Peucker: the points that keep a line's shape."""
    if len(points) < 3:
        return points
    (x1, y1), (x2, y2) = points[0], points[-1]
    dx, dy = x2 - x1, y2 - y1
    norm = math.hypot(dx, dy) or 1e-9
    far, at = 0.0, 0
    for i in range(1, len(points) - 1):
        x, y = points[i]
        d = abs(dy * x - dx * y + x2 * y1 - y2 * x1) / norm
        if d > far:
            far, at = d, i
    if far <= tol:
        return [points[0], points[-1]]
    return dp(points[:at + 1], tol)[:-1] + dp(points[at:], tol)


def main(src):
    data = json.load(open(src, encoding="utf-8"))
    out = []
    for f in data["features"]:
        p = f["properties"]
        code = p.get("ISO_A2_EH") or p.get("ISO_A2") or ""
        if code == "-99":
            code = ""
        geom = f["geometry"]
        polys = geom["coordinates"] if geom["type"] == "MultiPolygon" else [geom["coordinates"]]
        rings = []
        for poly in polys:
            ring = [(round(x, 1), round(y, 1)) for x, y in poly[0]]
            # A closed ring: simplified in two halves, split at its farthest point.
            k = max(range(len(ring)), key=lambda i: math.hypot(ring[i][0] - ring[0][0], ring[i][1] - ring[0][1]))
            ring = dp(ring[:k + 1], 0.35)[:-1] + dp(ring[k:], 0.35)
            if len(ring) >= 4:
                rings.append([v for pt in ring for v in pt])
        if not rings:
            continue
        out.append({
            "n": p.get("NAME") or p.get("ADMIN"),
            "c": p.get("CONTINENT", ""),
            "e": CONTINENT_EMOJI.get(p.get("CONTINENT", ""), "🌐"),
            "f": code,
            "p": int(p.get("POP_EST") or 0),
            "cap": CAPITALS.get(code, ""),
            "fun": FACTS.get(code, ""),
            "r": rings,
        })
    dest = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "profile", "airootfs", "usr", "local", "share", "ollie", "world.json")
    with open(dest, "w", encoding="utf-8") as fh:
        json.dump(out, fh, ensure_ascii=False, separators=(",", ":"))
    print(len(out), "countries,", os.path.getsize(dest), "bytes,", sum(len(r) // 2 for c in out for r in c["r"]), "points")


if __name__ == "__main__":
    main(sys.argv[1])
