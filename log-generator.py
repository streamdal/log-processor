import json
import time
import random
import socket

# Logstash agent host and port
logstash_host = "localhost"  # Replace with the actual host of Logstash agent
logstash_port = 5044

def generate_log():
    """Generates a simulated JSON log entry."""
    log_entry = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "user_id": random.randint(1, 100),
        "activity": random.choice(["login", "logout", "purchase", "view"]),
        "details": {
            "ip_address": f"192.168.1.{random.randint(1, 255)}",
            "location": random.choice(["USA", "Canada", "UK", "Germany", "France"])
        }
    }
    return json.dumps(log_entry)

def main():
    while True:
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
                sock.connect((logstash_host, logstash_port))
                while True:
                    log = generate_log()
                    print(log)
                    sock.sendall((log + "\n").encode("utf-8"))
                    time.sleep(1)  # Log every 1 second. Adjust as needed.
        except BrokenPipeError:
            print("Connection lost. Attempting to reconnect...")
            time.sleep(5)  # Wait for 5 seconds before reconnecting
        except Exception as e:
            print(f"An error occurred: {e}")
            break

if __name__ == "__main__":
    main()
