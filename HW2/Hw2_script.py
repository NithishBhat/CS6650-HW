import requests

# Your two instances
EC2_URL1 = "http://52.55.168.9:8080/albums"
EC2_URL2 = "http://54.145.29.135:8080/albums"

POST_DATA = {
    "id": "4",
    "title": "The Modern Sound of Betty Carter",
    "artist": "Betty Carter",
    "price": 49.99
}
HEADERS = {"Content-Type": "application/json"}

def data_from_both(url1, url2):
    try:
        print(f"Checking Instance 1...")
        r1 = requests.get(url1)
        print(f"Instance 1 has {len(r1.json())} albums.")
        
        print(f"Checking Instance 2...")
        r2 = requests.get(url2)
        print(f"Instance 2 has {len(r2.json())} albums.")
    except Exception as e:
        print(f"Error: {e}")

print("--- Initial State ---")
data_from_both(EC2_URL1, EC2_URL2)

print("\n--- Adding album to Instance 2 ONLY ---")
requests.post(EC2_URL2, json=POST_DATA, headers=HEADERS)

print("\n--- Final State ---")
data_from_both(EC2_URL1, EC2_URL2)