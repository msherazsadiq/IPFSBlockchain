import sys
import heapq

def dijkstra(file_path, start_node):
    # Read the graph from the file
    graph = {}
    with open(file_path, 'r') as file:
        for line in file:
            parts = line.strip().split()
            if len(parts) < 3:
                continue
            node1, node2, weight = parts[0], parts[1], int(parts[2])
            if node1 not in graph:
                graph[node1] = []
            if node2 not in graph:
                graph[node2] = []
            graph[node1].append((node2, weight))
            graph[node2].append((node1, weight))  # Assuming an undirected graph

    # Initialize data structures
    distances = {node: float('inf') for node in graph}
    distances[start_node] = 0
    priority_queue = [(0, start_node)]  # (distance, node)

    while priority_queue:
        current_distance, current_node = heapq.heappop(priority_queue)

        # Skip if we found a better path already
        if current_distance > distances[current_node]:
            continue

        for neighbor, weight in graph[current_node]:
            distance = current_distance + weight

            # If a shorter path to the neighbor is found
            if distance < distances[neighbor]:
                distances[neighbor] = distance
                heapq.heappush(priority_queue, (distance, neighbor))

    return distances

# Example usage
if __name__ == "__main__":


    file_path = sys.argv[1]
    start_node = 'A'

    try:
        distances = dijkstra(file_path, start_node)
        print("Shortest distances from node", start_node, ":")
        for node, distance in distances.items():
            print(f"{node}: {distance}")
    except Exception as e:
        print("Error:", e)
