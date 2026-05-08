import pandas as pd
import matplotlib.pyplot as plt

def plot_tab_file(file_path, column_labels, output_file):
    # Read the data from the tab-separated file
    data = pd.read_csv(file_path, sep='\t', header=None)

    # Extract x-axis data (first column) and other columns for comparison
    x_data = data.iloc[:, 0]
    y_data = data.iloc[:, 1:]
    
    # Define custom colors for each label 
    custom_colors = ['blue', 'red', 'yellow', 'green']
    custom_markers = ['o', 's', '^', 'D']

    # Plotting the data for each column with custom colors and markers
    plt.figure(figsize=(10, 6))
    for i, label in enumerate(column_labels[1:]):  # Skip the first label (x-axis label)
        plt.plot(x_data, y_data.iloc[:, i], label=label, color=custom_colors[i], marker=custom_markers[i])

    # Add labels and legend to the plot
    plt.xlabel(column_labels[0])  # Use the first label as x-axis label
    plt.ylabel('p99.9 slowdown')
    plt.legend()
    
    # Limit y-axis from 0 to 200
    plt.ylim(0, 200)

    # Save the plot as an EPS file
    plt.savefig(output_file, format='eps')

    # Show the plot
    plt.show()

if __name__ == "__main__":
    # Replace the file path with the actual path of your tab-separated file
    file_path = "final.csv"

    # Replace the column_labels with the actual labels for all columns
    column_labels = ['Load (fraction of max)', 'Single queue (no preemption)', 'Precise preemption: N(5,0)', 'Preemption with variance: N(5,1)', 'Preemption with variance: N(5,2)']

    # Replace "output.eps" with the desired output file name
    output_file = "fig5.eps"

    plot_tab_file(file_path, column_labels, output_file)